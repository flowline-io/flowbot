package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/flc1125/go-cron/v4"

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
	} else {
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

	method := wh.Method
	if method == "" {
		method = "POST"
	}
	method = strings.ToUpper(method)
	if !allowedWebhookMethods[method] {
		return nil, fmt.Errorf("pipeline %s: unsupported webhook method %q", name, wh.Method)
	}

	if wh.Auth == nil || (wh.Auth.Token == "" && wh.Auth.HMACSecret == "") {
		return nil, fmt.Errorf("pipeline %s: webhook trigger requires at least one of auth.token or auth.hmac_secret", name)
	}

	payload := wh.Payload
	if payload == "" {
		payload = config.WebhookPayloadRaw
	}
	if payload != config.WebhookPayloadRaw && payload != config.WebhookPayloadMapped {
		return nil, fmt.Errorf("pipeline %s: invalid webhook payload mode %q", name, wh.Payload)
	}

	eventType := wh.EventType
	if eventType == "" {
		eventType = "webhook." + wh.Path
	}

	hmacHeader := "X-Hub-Signature-256"
	tokenHeader := "X-Webhook-Token"
	if wh.Auth != nil {
		if wh.Auth.HMACHeader != "" {
			hmacHeader = wh.Auth.HMACHeader
		}
		if wh.Auth.TokenHeader != "" {
			tokenHeader = wh.Auth.TokenHeader
		}
	}

	wc := &WebhookConfig{
		Path:      wh.Path,
		Method:    method,
		Payload:   payload,
		EventType: eventType,
	}
	if wh.Auth != nil {
		wc.Auth = WebhookAuthConfig{
			Token:       wh.Auth.Token,
			HMACSecret:  wh.Auth.HMACSecret,
			HMACHeader:  hmacHeader,
			TokenHeader: tokenHeader,
		}
	}
	return wc, nil
}

func validateCronExpr(spec string) error {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := p.Parse(spec)
	return err
}
