package pipeline

import (
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type Definition struct {
	Name        string
	Description string
	Enabled     bool
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
			Trigger:     Trigger{Event: p.Trigger.Event},
		}
		for _, s := range p.Steps {
			d.Steps = append(d.Steps, Step{
				Name:       s.Name,
				Capability: hub.CapabilityType(s.Capability),
				Operation:  s.Operation,
				Params:     s.Params,
			})
		}
		defs = append(defs, d)
	}
	return defs
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
