package pipeline

import (
	"fmt"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff"
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
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)

	assert.Equal(t, "entity-123", rendered["entity"])
	assert.Equal(t, "https://example.com", rendered["link"])
	assert.Equal(t, "Hello World", rendered["title"])
}

func TestRenderContext_RenderParams_NoTemplates(t *testing.T) {
	rc := NewRenderContext(types.DataEvent{})
	params := map[string]any{"key": "value", "num": 42}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
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
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)

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
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	nested := rendered["nested"].(map[string]any)
	assert.Equal(t, "123", nested["inner"])
}

func TestRenderContext_RenderParams_StringSlice(t *testing.T) {
	event := types.DataEvent{EventID: "evt1", EntityID: "eid"}
	rc := NewRenderContext(event)

	params := map[string]any{
		"items": []any{"{{event.id}}", "static"},
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	items := rendered["items"].([]any)
	assert.Equal(t, "eid", items[0])
	assert.Equal(t, "static", items[1])
}

func TestRenderContext_RenderParams_MissingEventField(t *testing.T) {
	rc := NewRenderContext(types.DataEvent{})
	rendered, err := rc.RenderParams(map[string]any{"ref": "{{event.url}}"})
	require.NoError(t, err)
	assert.Equal(t, "", rendered["ref"])
}

func TestRenderContext_RenderParams_NonStringEventField(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"url": 42},
	}
	rc := NewRenderContext(event)
	rendered, err := rc.RenderParams(map[string]any{"ref": "{{event.url}}"})
	require.NoError(t, err)
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
	rendered, err := rc.RenderParams(map[string]any{"ref": "{{json (event \"url\")}}"})
	require.NoError(t, err)
	assert.Equal(t, `{"href":"https://x.com"}`, rendered["ref"])
}

func TestRenderContext_RenderParams_Condition(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"status": "done"},
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"action": "{{if eq .Event.status \"done\"}}archive{{else}}skip{{end}}",
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	assert.Equal(t, "archive", rendered["action"])
}

func TestRenderContext_RenderParams_ConditionElse(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"status": "pending"},
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"action": "{{if eq .Event.status \"done\"}}archive{{else}}skip{{end}}",
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	assert.Equal(t, "skip", rendered["action"])
}

func TestRenderContext_RenderParams_Loop(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"tags": []any{"a", "b", "c"}},
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"joined": "{{range .Event.tags}}{{.}}-{{end}}",
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	assert.Equal(t, "a-b-c-", rendered["joined"])
}

func TestRenderContext_RenderParams_LoopElse(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"tags": []any{}},
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"result": "{{range .Event.tags}}x{{else}}empty{{end}}",
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	assert.Equal(t, "empty", rendered["result"])
}

func TestRenderContext_RenderParams_NestedConditionLoop(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"items": []any{"a", "", "c"}},
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"filtered": "{{range .Event.items}}{{if .}}{{.}},{{end}}{{end}}",
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	assert.Equal(t, "a,c,", rendered["filtered"])
}

func TestRenderContext_RenderParams_Default(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{},
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"label": "{{default \"unknown\" .Event.category}}",
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	assert.Equal(t, "unknown", rendered["label"])
}

func TestRenderContext_RenderParams_JoinAndContains(t *testing.T) {
	event := types.DataEvent{
		Data: types.KV{"tags": []any{"alpha", "beta", "gamma"}},
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"csv": "{{join .Event.tags \",\"}}",
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	assert.Equal(t, "alpha,beta,gamma", rendered["csv"])
}

func TestRenderContext_RenderParams_InvalidTemplate(t *testing.T) {
	rc := NewRenderContext(types.DataEvent{})
	_, err := rc.RenderParams(map[string]any{"bad": "{{if xxx}}"})
	require.Error(t, err)
}

func TestRenderContext_RenderString(t *testing.T) {
	event := types.DataEvent{
		EntityID: "123",
		Data:     types.KV{"url": "https://example.com"},
	}
	rc := NewRenderContext(event)

	result, err := rc.RenderString("id={{event.id}} url={{event.url}}")
	require.NoError(t, err)
	assert.Equal(t, "id=123 url=https://example.com", result)
}

func TestRenderContext_RenderString_NewSyntax(t *testing.T) {
	event := types.DataEvent{
		EntityID: "123",
	}
	rc := NewRenderContext(event)

	result, err := rc.RenderString("{{if .Event.id}}ID:{{.Event.id}}{{end}}")
	require.NoError(t, err)
	assert.Equal(t, "ID:123", result)
}

func TestRenderContext_RenderParams_EventTopLevelFields(t *testing.T) {
	event := types.DataEvent{
		EventID:   "evt-001",
		EventType: "bookmark.created",
		EntityID:  "entity-123",
		Source:    "test",
	}
	rc := NewRenderContext(event)

	params := map[string]any{
		"event_id":   "{{event.event_id}}",
		"event_type": "{{event.event_type}}",
		"id":         "{{event.id}}",
		"entity_id":  "{{event.entity_id}}",
		"source":     "{{event.source}}",
	}
	rendered, err := rc.RenderParams(params)
	require.NoError(t, err)
	assert.Equal(t, "evt-001", rendered["event_id"])
	assert.Equal(t, "bookmark.created", rendered["event_type"])
	assert.Equal(t, "entity-123", rendered["id"])
	assert.Equal(t, "entity-123", rendered["entity_id"])
	assert.Equal(t, "test", rendered["source"])
}

func TestBuildBackoff_NoConfig(t *testing.T) {
	bo := (&types.RetryConfig{}).BuildBackOff()
	require.NotNil(t, bo)
}

func TestBuildBackoff_Exponential(t *testing.T) {
	cfg := &types.RetryConfig{
		MaxAttempts: 3,
		Delay:       1 * time.Second,
		Backoff:     types.BackoffExponential,
		MaxDelay:    10 * time.Second,
	}
	bo := cfg.BuildBackOff()
	require.NotNil(t, bo)
	assert.NotEqual(t, backoff.Stop, bo.NextBackOff())
}

func TestBuildBackoff_Fixed(t *testing.T) {
	cfg := &types.RetryConfig{
		MaxAttempts: 3,
		Delay:       500 * time.Millisecond,
		Backoff:     types.BackoffFixed,
	}
	bo := cfg.BuildBackOff()
	require.NotNil(t, bo)
	delay := bo.NextBackOff()
	assert.Equal(t, cfg.Delay, delay)
}

func TestBuildBackoff_Linear(t *testing.T) {
	cfg := &types.RetryConfig{
		MaxAttempts: 3,
		Delay:       1 * time.Second,
		Backoff:     types.BackoffLinear,
		MaxDelay:    30 * time.Second,
	}
	bo := cfg.BuildBackOff()
	require.NotNil(t, bo)
	assert.NotEqual(t, backoff.Stop, bo.NextBackOff())
}

func TestBuildBackoff_WithJitter(t *testing.T) {
	cfg := &types.RetryConfig{
		MaxAttempts: 3,
		Delay:       1 * time.Second,
		Backoff:     types.BackoffExponential,
		Jitter:      true,
	}
	bo := cfg.BuildBackOff()
	require.NotNil(t, bo)
	assert.NotEqual(t, backoff.Stop, bo.NextBackOff())
}

func TestIsRetryable_NoFilter(t *testing.T) {
	cfg := &types.RetryConfig{}
	err := fmt.Errorf("generic error")
	assert.True(t, isRetryable(err, cfg))
}

func TestIsRetryable_NilConfig(t *testing.T) {
	err := fmt.Errorf("generic error")
	assert.True(t, isRetryable(err, nil))
}

func TestIsRetryable_WithRetryOnMatch(t *testing.T) {
	cfg := &types.RetryConfig{
		RetryOn: []string{"timeout", "rate_limited"},
	}
	te := &types.Error{Code: "timeout", Retryable: true}
	assert.True(t, isRetryable(te, cfg))
}

func TestIsRetryable_WithRetryOnNoMatch(t *testing.T) {
	cfg := &types.RetryConfig{
		RetryOn: []string{"timeout"},
	}
	err := fmt.Errorf("some other error")
	assert.False(t, isRetryable(err, cfg))
}

func TestIsRetryable_RetryableFlag(t *testing.T) {
	cfg := &types.RetryConfig{
		RetryOn: []string{"timeout"},
	}
	te := &types.Error{Retryable: true}
	assert.True(t, isRetryable(te, cfg))
}

func TestRetryConfig_RetryEnabled(t *testing.T) {
	assert.False(t, (*types.RetryConfig)(nil).RetryEnabled())
	assert.False(t, (&types.RetryConfig{}).RetryEnabled())
	assert.True(t, (&types.RetryConfig{MaxAttempts: 3}).RetryEnabled())
}

func TestCheckpointData_Marshaling(t *testing.T) {
	cp := &CheckpointData{
		StepIndex: 2,
		StepResults: map[string]*StepResult{
			"step1": {Name: "step1", Output: map[string]any{"id": "123"}},
		},
		HeartbeatAt: time.Now(),
	}
	data, err := sonic.Marshal(cp)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var restored CheckpointData
	err = sonic.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, 2, restored.StepIndex)
	assert.Equal(t, "step1", restored.StepResults["step1"].Name)
}

func TestBuildStepResults(t *testing.T) {
	event := types.DataEvent{EventID: "evt-1"}
	rc := NewRenderContext(event)
	rc.RecordStepResult("step1", map[string]any{"id": "123"})
	rc.RecordStepResult("step2", map[string]any{"id": "456"})

	results := buildStepResults(rc)
	assert.Len(t, results, 2)
	assert.Equal(t, "123", results["step1"].Output["id"])
	assert.Equal(t, "456", results["step2"].Output["id"])
}
