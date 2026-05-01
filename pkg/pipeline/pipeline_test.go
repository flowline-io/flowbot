package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Basic(t *testing.T) {
	cfg := []config.Pipeline{
		{
			Name:        "my-pipeline",
			Description: "A test pipeline",
			Enabled:     true,
			Trigger:     config.PipelineTrigger{Event: "bookmark.created"},
			Steps: []config.PipelineStep{
				{Name: "step1", Capability: "bookmark", Operation: "list", Params: map[string]any{"limit": 10}},
			},
		},
	}
	defs := LoadConfig(cfg)
	require.Len(t, defs, 1)
	assert.Equal(t, "my-pipeline", defs[0].Name)
	assert.Equal(t, "A test pipeline", defs[0].Description)
	assert.True(t, defs[0].Enabled)
	assert.Equal(t, "bookmark.created", defs[0].Trigger.Event)
	require.Len(t, defs[0].Steps, 1)
	assert.Equal(t, "step1", defs[0].Steps[0].Name)
	assert.Equal(t, hub.CapabilityType("bookmark"), defs[0].Steps[0].Capability)
	assert.Equal(t, "list", defs[0].Steps[0].Operation)
	assert.Equal(t, 10, defs[0].Steps[0].Params["limit"])
}

func TestLoadConfig_DisabledSkipped(t *testing.T) {
	cfg := []config.Pipeline{
		{Name: "enabled", Enabled: true},
		{Name: "disabled", Enabled: false},
	}
	defs := LoadConfig(cfg)
	require.Len(t, defs, 1)
	assert.Equal(t, "enabled", defs[0].Name)
}

func TestLoadConfig_MultipleSteps(t *testing.T) {
	cfg := []config.Pipeline{
		{
			Name:    "multi-step",
			Enabled: true,
			Steps: []config.PipelineStep{
				{Name: "s1", Capability: "bookmark", Operation: "list"},
				{Name: "s2", Capability: "archive", Operation: "add"},
				{Name: "s3", Capability: "kanban", Operation: "create_task"},
			},
		},
	}
	defs := LoadConfig(cfg)
	require.Len(t, defs, 1)
	assert.Len(t, defs[0].Steps, 3)
}

func TestLoadConfig_EmptySteps(t *testing.T) {
	cfg := []config.Pipeline{
		{Name: "empty-steps", Enabled: true},
	}
	defs := LoadConfig(cfg)
	require.Len(t, defs, 1)
	assert.Empty(t, defs[0].Steps)
}

func TestLoadConfig_Empty(t *testing.T) {
	defs := LoadConfig(nil)
	assert.Empty(t, defs)
}

func TestDefinition_FindByEvent_Match(t *testing.T) {
	d := Definition{Name: "p", Trigger: Trigger{Event: "bookmark.created"}}
	matched := d.FindByEvent("bookmark.created")
	require.Len(t, matched, 1)
	assert.Equal(t, "p", matched[0].Name)
}

func TestDefinition_FindByEvent_NoMatch(t *testing.T) {
	d := Definition{Name: "p", Trigger: Trigger{Event: "bookmark.created"}}
	matched := d.FindByEvent("archive.created")
	assert.Empty(t, matched)
}

func TestFindByEvent_MultipleMatches(t *testing.T) {
	defs := []Definition{
		{Name: "p1", Trigger: Trigger{Event: "e1"}},
		{Name: "p2", Trigger: Trigger{Event: "e2"}},
		{Name: "p3", Trigger: Trigger{Event: "e1"}},
	}
	matched := FindByEvent(defs, "e1")
	require.Len(t, matched, 2)
}

func TestFindByEvent_NoMatches(t *testing.T) {
	defs := []Definition{
		{Name: "p1", Trigger: Trigger{Event: "e1"}},
	}
	matched := FindByEvent(defs, "nonexistent")
	assert.Empty(t, matched)
}

func TestFindByEvent_EmptySlice(t *testing.T) {
	matched := FindByEvent(nil, "e1")
	assert.Empty(t, matched)
}

func TestNewRenderContext(t *testing.T) {
	event := types.DataEvent{EventID: "evt1", EventType: "bookmark.created", EntityID: "123"}
	rc := NewRenderContext(event)
	assert.Equal(t, "evt1", rc.Event.EventID)
	assert.NotNil(t, rc.Steps)
	assert.Empty(t, rc.Steps)
}

func TestRenderContext_RecordStepResult(t *testing.T) {
	rc := NewRenderContext(types.DataEvent{})
	rc.RecordStepResult("step1", map[string]any{"id": "abc", "url": "https://x.com"})
	assert.Contains(t, rc.Steps, "step1")
	assert.Equal(t, "abc", rc.Steps["step1"]["id"])
}

func TestRenderContext_RenderParams_EventFields(t *testing.T) {
	event := types.DataEvent{
		EventID:  "evt1",
		EntityID: "entity-123",
		Data:     types.KV{"url": "https://example.com", "title": "Hello World"},
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"entity": "{{event.id}}",
		"link":   "{{event.url}}",
		"title":  "{{event.title}}",
	}
	rendered := rc.RenderParams(params)

	assert.Equal(t, "entity-123", rendered["entity"])
	assert.Equal(t, "https://example.com", rendered["link"])
	assert.Equal(t, "Hello World", rendered["title"])
}

func TestRenderContext_RenderParams_NoTemplates(t *testing.T) {
	rc := NewRenderContext(types.DataEvent{})
	params := map[string]any{"key": "value", "num": 42}
	rendered := rc.RenderParams(params)
	assert.Equal(t, "value", rendered["key"])
	assert.Equal(t, 42, rendered["num"])
}

func TestRenderContext_RenderParams_StepReferences(t *testing.T) {
	rc := NewRenderContext(types.DataEvent{})
	rc.RecordStepResult("archive", map[string]any{
		"id":  "archive-1",
		"url": "https://archived.example.com",
	})

	params := map[string]any{
		"ref_id":  "{{steps.archive.id}}",
		"ref_url": "{{steps.archive.url}}",
	}
	rendered := rc.RenderParams(params)

	assert.Equal(t, "archive-1", rendered["ref_id"])
	assert.Equal(t, "https://archived.example.com", rendered["ref_url"])
}

func TestRenderContext_RenderParams_NestedMap(t *testing.T) {
	event := types.DataEvent{EventID: "evt1", EntityID: "123"}
	rc := NewRenderContext(event)

	params := map[string]any{
		"nested": map[string]any{
			"inner": "{{event.id}}",
		},
	}
	rendered := rc.RenderParams(params)
	nested := rendered["nested"].(map[string]any)
	assert.Equal(t, "123", nested["inner"])
}

func TestRenderContext_RenderParams_StringSlice(t *testing.T) {
	event := types.DataEvent{EventID: "evt1", EntityID: "eid"}
	rc := NewRenderContext(event)

	params := map[string]any{
		"items": []any{"{{event.id}}", "static"},
	}
	rendered := rc.RenderParams(params)
	items := rendered["items"].([]any)
	assert.Equal(t, "eid", items[0])
	assert.Equal(t, "static", items[1])
}

func TestRenderContext_RenderParams_MissingEventField(t *testing.T) {
	rc := NewRenderContext(types.DataEvent{})
	rendered := rc.RenderParams(map[string]any{"ref": "{{event.url}}"})
	assert.Equal(t, "", rendered["ref"])
}

func TestRenderContext_RenderParams_NonStringEventField(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"url": 42},
	}
	rc := NewRenderContext(event)
	rendered := rc.RenderParams(map[string]any{"ref": "{{event.url}}"})
	assert.Equal(t, "42", rendered["ref"])
}

func TestNewEngine(t *testing.T) {
	e := NewEngine(nil, nil)
	assert.NotNil(t, e)
	assert.NotNil(t, e.Handler())
}

func TestConvertToTypesKV(t *testing.T) {
	m := map[string]any{"a": 1, "b": "x"}
	result := convertToTypesKV(m)
	assert.Equal(t, 1, result["a"])
	assert.Equal(t, "x", result["b"])
	assert.Len(t, result, 2)
}

func TestConvertToTypesKV_Empty(t *testing.T) {
	result := convertToTypesKV(map[string]any{})
	assert.Empty(t, result)
}

func TestRenderContext_RenderParams_JSONField(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"url": map[string]any{"href": "https://x.com"}},
	}
	rc := NewRenderContext(event)
	rendered := rc.RenderParams(map[string]any{"ref": "{{event.url}}"})
	// JSON-encoded version of the map
	var decoded any
	_ = json.Unmarshal([]byte(rendered["ref"].(string)), &decoded)
	assert.NotNil(t, decoded)
}
