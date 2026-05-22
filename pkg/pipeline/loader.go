package pipeline

import (
	"fmt"
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
	Event      string
	Cron       string
	CronTimeout time.Duration
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
		d := Definition{
			Name:        p.Name,
			Description: p.Description,
			Enabled:     p.Enabled,
			Resumable:   p.Resumable,
			Trigger:     cronTrigger(p.Name, p.Trigger),
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

func cronTrigger(name string, cfg config.PipelineTrigger) Trigger {
	t := Trigger{Event: cfg.Event, Cron: cfg.Cron}
	if cfg.CronTimeout != "" {
		d, err := time.ParseDuration(cfg.CronTimeout)
		if err != nil {
			flog.Error(fmt.Errorf("pipeline %s: invalid cron_timeout %q: %w", name, cfg.CronTimeout, err))
			return t
		}
		t.CronTimeout = d
	} else {
		t.CronTimeout = 10 * time.Minute
	}
	return t
}

func validateCronExpr(spec string) error {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := p.Parse(spec)
	return err
}
