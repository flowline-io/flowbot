package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flc1125/go-cron/v4"
	"gopkg.in/yaml.v3"

	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type Definition struct {
	Name        string
	Description string
	Enabled     bool
	Resumable   bool
	Trigger     Trigger
	Steps       []Step
	ParentName  string
}

type Trigger struct {
	Event       string
	Cron        string
	CronTimeout time.Duration
	Webhook     *WebhookConfig
}

type WebhookConfig struct {
	Path      string
	Method    string
	Auth      WebhookAuthConfig
	Payload   config.WebhookPayloadMode
	EventType string
}

type WebhookAuthConfig struct {
	Token       string
	HMACSecret  string
	HMACHeader  string
	TokenHeader string
}

type Step struct {
	Name       string
	Capability hub.CapabilityType
	Operation  string
	Params     map[string]any
	Retry      *backoff.Config
}

func LoadConfig(cfg []config.Pipeline) []Definition {
	defs := make([]Definition, 0, len(cfg))
	for _, p := range cfg {
		if !p.Enabled {
			continue
		}
		if p.Trigger.Cron != "" {
			if err := validateCronExpr(p.Trigger.Cron); err != nil {
				flog.Error(fmt.Errorf("pipeline %s: invalid cron expression %q: %w", p.Name, p.Trigger.Cron, err))
				continue
			}
		}
		trigger, err := convertTrigger(p.Name, p.Trigger)
		if err != nil {
			flog.Error(err)
			continue
		}
		d := Definition{
			Name:        p.Name,
			Description: p.Description,
			Enabled:     p.Enabled,
			Resumable:   p.Resumable,
			Trigger:     trigger,
		}
		for _, s := range p.Steps {
			retry, err := convertRetryConfig(s.Retry)
			if err != nil {
				flog.Error(fmt.Errorf("pipeline %s step %s: invalid retry config: %w", p.Name, s.Name, err))
				continue
			}
			d.Steps = append(d.Steps, Step{
				Name:       s.Name,
				Capability: hub.CapabilityType(s.Capability),
				Operation:  s.Operation,
				Params:     s.Params,
				Retry:      retry,
			})
		}
		defs = append(defs, d)
	}
	return defs
}

func convertRetryConfig(cfg *config.PipelineStepRetry) (*backoff.Config, error) {
	if cfg == nil || cfg.MaxAttempts <= 0 {
		return nil, nil
	}
	delay, err := time.ParseDuration(cfg.Delay)
	if err != nil && cfg.Delay != "" {
		return nil, fmt.Errorf("invalid delay %q: %w", cfg.Delay, err)
	}
	maxDelay, err := time.ParseDuration(cfg.MaxDelay)
	if err != nil && cfg.MaxDelay != "" {
		return nil, fmt.Errorf("invalid max_delay %q: %w", cfg.MaxDelay, err)
	}
	multiplier := 2.0
	switch cfg.Backoff {
	case "fixed", "linear":
		multiplier = 1.0
	}
	return &backoff.Config{
		MaxAttempts:     cfg.MaxAttempts,
		InitialInterval: delay,
		MaxInterval:     maxDelay,
		Multiplier:      multiplier,
		Jitter:          cfg.Jitter,
		RetryOn:         cfg.RetryOn,
	}, nil
}

func (d Definition) FindByEvent(eventType string) []Definition {
	if d.Trigger.Event == eventType {
		return []Definition{d}
	}
	return nil
}

func FindByEvent(defs []Definition, eventType string) []Definition {
	var matched []Definition
	for _, d := range defs {
		if d.Trigger.Event == eventType {
			matched = append(matched, d)
		}
	}
	return matched
}

func convertTrigger(name string, cfg config.PipelineTrigger) (Trigger, error) {
	t := Trigger{Event: cfg.Event, Cron: cfg.Cron}

	if cfg.CronTimeout != "" {
		d, err := time.ParseDuration(cfg.CronTimeout)
		if err != nil {
			flog.Error(fmt.Errorf("pipeline %s: invalid cron_timeout %q: %w", name, cfg.CronTimeout, err))
		} else {
			t.CronTimeout = d
		}
	} else if cfg.Cron != "" {
		t.CronTimeout = 10 * time.Minute
	}

	if cfg.Webhook != nil && (cfg.Event != "" || cfg.Cron != "") {
		return t, fmt.Errorf("pipeline %s: webhook trigger cannot be combined with event or cron", name)
	}

	wh, err := convertWebhookTrigger(name, cfg.Webhook)
	if err != nil {
		return t, err
	}
	t.Webhook = wh

	return t, nil
}

var allowedWebhookMethods = map[string]bool{
	"GET":  true,
	"POST": true,
	"PUT":  true,
}

func convertWebhookTrigger(name string, wh *config.WebhookTrigger) (*WebhookConfig, error) {
	if wh == nil {
		return nil, nil
	}

	if wh.Path == "" {
		return nil, fmt.Errorf("pipeline %s: webhook trigger path must not be empty", name)
	}

	method, err := validateWebhookMethod(name, wh.Method)
	if err != nil {
		return nil, err
	}

	if err := validateWebhookAuth(name, wh.Auth); err != nil {
		return nil, err
	}

	payload, err := validateWebhookPayload(name, wh.Payload)
	if err != nil {
		return nil, err
	}

	eventType := wh.EventType
	if eventType == "" {
		eventType = "webhook." + wh.Path
	}

	return &WebhookConfig{
		Path:      wh.Path,
		Method:    method,
		Auth:      buildWebhookAuthConfig(wh.Auth),
		Payload:   payload,
		EventType: eventType,
	}, nil
}

func validateWebhookMethod(name, method string) (string, error) {
	if method == "" {
		return "POST", nil
	}
	m := strings.ToUpper(method)
	if !allowedWebhookMethods[m] {
		return "", fmt.Errorf("pipeline %s: unsupported webhook method %q", name, method)
	}
	return m, nil
}

func validateWebhookAuth(name string, auth *config.WebhookAuth) error {
	if auth == nil || (auth.Token == "" && auth.HMACSecret == "") {
		return fmt.Errorf("pipeline %s: webhook trigger requires at least one of auth.token or auth.hmac_secret", name)
	}
	return nil
}

func validateWebhookPayload(name string, payload config.WebhookPayloadMode) (config.WebhookPayloadMode, error) {
	if payload == "" {
		return config.WebhookPayloadRaw, nil
	}
	if payload != config.WebhookPayloadRaw && payload != config.WebhookPayloadMapped {
		return "", fmt.Errorf("pipeline %s: invalid webhook payload mode %q", name, payload)
	}
	return payload, nil
}

func buildWebhookAuthConfig(auth *config.WebhookAuth) WebhookAuthConfig {
	if auth == nil {
		return WebhookAuthConfig{}
	}
	hmacHeader := auth.HMACHeader
	if hmacHeader == "" {
		hmacHeader = "X-Hub-Signature-256"
	}
	tokenHeader := auth.TokenHeader
	if tokenHeader == "" {
		tokenHeader = "X-Webhook-Token"
	}
	return WebhookAuthConfig{
		Token:       auth.Token,
		HMACSecret:  auth.HMACSecret,
		HMACHeader:  hmacHeader,
		TokenHeader: tokenHeader,
	}
}

func validateCronExpr(spec string) error {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := p.Parse(spec)
	return err
}

// ExpandDefinitions fans out an editor definition with multiple triggers into
// engine Definition instances with compound names to avoid key collisions.
func ExpandDefinitions(defs []EditorDefinition) []Definition {
	var expanded []Definition
	for _, d := range defs {
		if !d.Enabled {
			continue
		}
		for i, t := range d.Triggers {
			if !t.Enabled {
				continue
			}
			compoundName := fmt.Sprintf("%s__trigger_%s_%d", d.Name, t.Type, i)
			expanded = append(expanded, Definition{
				Name:        compoundName,
				Description: d.Description,
				Enabled:     true,
				Resumable:   d.Resumable,
				Trigger:     t.toEngineTrigger(),
				Steps:       d.Steps,
				ParentName:  d.Name,
			})
		}
	}
	return expanded
}

func (t TriggerEntry) toEngineTrigger() Trigger {
	tr := Trigger{}
	switch t.Type {
	case "event":
		tr.Event = t.Event
	case "cron":
		tr.Cron = t.Cron
		if t.CronTimeout != "" {
			d, err := time.ParseDuration(t.CronTimeout)
			if err != nil {
				flog.Error(fmt.Errorf("pipeline cron: invalid cron_timeout %q: %w", t.CronTimeout, err))
			} else {
				tr.CronTimeout = d
			}
		}
		if tr.CronTimeout == 0 {
			tr.CronTimeout = 10 * time.Minute
		}
	case "webhook":
		tr.Webhook = t.Webhook
	}
	return tr
}

// ParseEditorYAML parses a YAML string into an EditorDefinition.
func ParseEditorYAML(yamlStr string) (*EditorDefinition, error) {
	if yamlStr == "" {
		return nil, fmt.Errorf("parse editor yaml: empty input")
	}
	var def EditorDefinition
	if err := yaml.Unmarshal([]byte(yamlStr), &def); err != nil {
		return nil, fmt.Errorf("parse editor yaml: %w", err)
	}
	return &def, nil
}

// LoadFromDB loads published pipeline definitions from a DefinitionReader.
func LoadFromDB(ctx context.Context, reader DefinitionReader) ([]Definition, error) {
	if reader == nil {
		return nil, nil
	}
	records, err := reader.ListPublishedDefinitions(ctx)
	if err != nil {
		return nil, fmt.Errorf("load definitions from db: %w", err)
	}
	var allDefs []Definition
	for _, rec := range records {
		ed, err := ParseEditorYAML(rec.YAML)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", rec.Name, err)
		}
		allDefs = append(allDefs, ExpandDefinitions([]EditorDefinition{*ed})...)
	}
	return allDefs, nil
}
