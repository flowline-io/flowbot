package pipeline

import (
	"github.com/flowline-io/flowbot/pkg/pipeline/template"
	"github.com/flowline-io/flowbot/pkg/types"
)

type RenderContext struct {
	Event  types.DataEvent
	Steps  map[string]map[string]any
	Input  map[string]any
	engine *template.Engine
}

func NewRenderContext(event types.DataEvent) *RenderContext {
	return &RenderContext{
		Event:  event,
		Steps:  make(map[string]map[string]any),
		engine: template.New(),
	}
}

func (rc *RenderContext) RecordStepResult(stepName string, result map[string]any) {
	rc.Steps[stepName] = result
}

func (rc *RenderContext) RenderParams(params map[string]any) (map[string]any, error) {
	return rc.engine.Render(params, rc.templateData())
}

func (rc *RenderContext) RenderString(s string) (string, error) {
	return rc.engine.RenderString(s, rc.templateData())
}

func (rc *RenderContext) templateData() *template.TemplateData {
	event := make(map[string]any)
	if rc.Event.Data != nil {
		for k, v := range rc.Event.Data {
			event[k] = v
		}
	}
	event["id"] = rc.Event.EntityID
	event["event_id"] = rc.Event.EventID
	event["event_type"] = rc.Event.EventType
	event["source"] = rc.Event.Source
	event["capability"] = rc.Event.Capability
	event["operation"] = rc.Event.Operation
	event["backend"] = rc.Event.Backend
	event["app"] = rc.Event.App
	event["entity_id"] = rc.Event.EntityID
	event["idempotency_key"] = rc.Event.IdempotencyKey
	event["uid"] = rc.Event.UID
	event["topic"] = rc.Event.Topic

	return &template.TemplateData{
		Event: event,
		Steps: rc.Steps,
		Input: rc.Input,
	}
}
