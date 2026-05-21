package pipeline

import (
	"fmt"
	"time"

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
	Event string
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
		d := Definition{
			Name:        p.Name,
			Description: p.Description,
			Enabled:     p.Enabled,
			Resumable:   p.Resumable,
			Trigger:     Trigger{Event: p.Trigger.Event},
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
