package pipeline

import (
	"encoding/json"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types"
)

type RenderContext struct {
	Event types.DataEvent
	Steps map[string]map[string]any
}

func NewRenderContext(event types.DataEvent) *RenderContext {
	return &RenderContext{
		Event: event,
		Steps: make(map[string]map[string]any),
	}
}

func (rc *RenderContext) RecordStepResult(stepName string, result map[string]any) {
	rc.Steps[stepName] = result
}

func (rc *RenderContext) RenderParams(params map[string]any) map[string]any {
	rendered := make(map[string]any, len(params))
	for key, value := range params {
		rendered[key] = rc.renderValue(value)
	}
	return rendered
}

func (rc *RenderContext) renderValue(value any) any {
	switch v := value.(type) {
	case string:
		return rc.renderString(v)
	case map[string]any:
		return rc.RenderParams(v)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = rc.renderValue(item)
		}
		return result
	default:
		return v
	}
}

func (rc *RenderContext) renderString(s string) string {
	if !strings.Contains(s, "{{") {
		return s
	}

	result := strings.ReplaceAll(s, "{{event.id}}", rc.Event.EntityID)
	result = strings.ReplaceAll(result, "{{event.url}}", rc.getEventField("url"))
	result = strings.ReplaceAll(result, "{{event.title}}", rc.getEventField("title"))

	// Render step references like {{steps.some_step.field_name}}
	for stepName, stepResult := range rc.Steps {
		for field, fieldValue := range stepResult {
			placeholder := "{{steps." + stepName + "." + field + "}}"
			strVal, _ := json.Marshal(fieldValue)
			result = strings.ReplaceAll(result, placeholder, toString(strVal))
		}
	}

	return result
}

func (rc *RenderContext) getEventField(field string) string {
	if rc.Event.Data != nil {
		if v, ok := rc.Event.Data[field]; ok {
			switch val := v.(type) {
			case string:
				return val
			default:
				b, _ := json.Marshal(val)
				return toString(b)
			}
		}
	}
	return ""
}

func toString(b []byte) string {
	s := string(b)
	s = strings.Trim(s, `"`)
	return s
}
