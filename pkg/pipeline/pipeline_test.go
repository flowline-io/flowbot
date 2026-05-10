package pipeline

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  []config.Pipeline
		test func(t *testing.T, defs []Definition)
	}{
		{
			name: "basic",
			cfg: []config.Pipeline{
				{
					Name:        "my-pipeline",
					Description: "A test pipeline",
					Enabled:     true,
					Trigger:     config.PipelineTrigger{Event: "bookmark.created"},
					Steps: []config.PipelineStep{
						{Name: "step1", Capability: "bookmark", Operation: "list", Params: map[string]any{"limit": 10}},
					},
				},
			},
			test: func(t *testing.T, defs []Definition) {
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
			},
		},
		{
			name: "disabled-skipped",
			cfg: []config.Pipeline{
				{Name: "enabled", Enabled: true},
				{Name: "disabled", Enabled: false},
			},
			test: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Equal(t, "enabled", defs[0].Name)
			},
		},
		{
			name: "multiple-steps",
			cfg: []config.Pipeline{
				{
					Name:    "multi-step",
					Enabled: true,
					Steps: []config.PipelineStep{
						{Name: "s1", Capability: "bookmark", Operation: "list"},
						{Name: "s2", Capability: "archive", Operation: "add"},
						{Name: "s3", Capability: "kanban", Operation: "create_task"},
					},
				},
			},
			test: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Len(t, defs[0].Steps, 3)
			},
		},
		{
			name: "empty-steps",
			cfg: []config.Pipeline{
				{Name: "empty-steps", Enabled: true},
			},
			test: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Empty(t, defs[0].Steps)
			},
		},
		{
			name: "empty",
			cfg:  nil,
			test: func(t *testing.T, defs []Definition) {
				assert.Empty(t, defs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defs := LoadConfig(tt.cfg)
			tt.test(t, defs)
		})
	}
}

func TestFindByEvent(t *testing.T) {
	t.Run("method-match", func(t *testing.T) {
		d := Definition{Name: "p", Trigger: Trigger{Event: "bookmark.created"}}
		matched := d.FindByEvent("bookmark.created")
		require.Len(t, matched, 1)
		assert.Equal(t, "p", matched[0].Name)
	})

	t.Run("method-no-match", func(t *testing.T) {
		d := Definition{Name: "p", Trigger: Trigger{Event: "bookmark.created"}}
		matched := d.FindByEvent("archive.created")
		assert.Empty(t, matched)
	})

	t.Run("package-multiple-matches", func(t *testing.T) {
		defs := []Definition{
			{Name: "p1", Trigger: Trigger{Event: "e1"}},
			{Name: "p2", Trigger: Trigger{Event: "e2"}},
			{Name: "p3", Trigger: Trigger{Event: "e1"}},
		}
		matched := FindByEvent(defs, "e1")
		require.Len(t, matched, 2)
	})

	t.Run("package-no-matches", func(t *testing.T) {
		defs := []Definition{
			{Name: "p1", Trigger: Trigger{Event: "e1"}},
		}
		matched := FindByEvent(defs, "nonexistent")
		assert.Empty(t, matched)
	})

	t.Run("package-empty-slice", func(t *testing.T) {
		matched := FindByEvent(nil, "e1")
		assert.Empty(t, matched)
	})
}

func TestRenderContext(t *testing.T) {
	t.Run("new-render-context", func(t *testing.T) {
		event := types.DataEvent{EventID: "evt1", EventType: "bookmark.created", EntityID: "123"}
		rc := NewRenderContext(event)
		assert.Equal(t, "evt1", rc.Event.EventID)
		assert.NotNil(t, rc.Steps)
		assert.Empty(t, rc.Steps)
	})

	t.Run("record-step-result", func(t *testing.T) {
		rc := NewRenderContext(types.DataEvent{})
		rc.RecordStepResult("step1", map[string]any{"id": "abc", "url": "https://x.com"})
		assert.Contains(t, rc.Steps, "step1")
		assert.Equal(t, "abc", rc.Steps["step1"]["id"])
	})

	t.Run("build-step-results", func(t *testing.T) {
		event := types.DataEvent{EventID: "evt-1"}
		rc := NewRenderContext(event)
		rc.RecordStepResult("step1", map[string]any{"id": "123"})
		rc.RecordStepResult("step2", map[string]any{"id": "456"})

		results := buildStepResults(rc)
		assert.Len(t, results, 2)
		assert.Equal(t, "123", results["step1"].Output["id"])
		assert.Equal(t, "456", results["step2"].Output["id"])
	})
}

func TestRenderParams(t *testing.T) {
	tests := []struct {
		name        string
		event       types.DataEvent
		recordSteps map[string]map[string]any
		params      map[string]any
		asserts     func(t *testing.T, rendered map[string]any)
		assertErr   string
	}{
		{
			name: "event-fields",
			event: types.DataEvent{
				EventID:  "evt1",
				EntityID: "entity-123",
				Data:     types.KV{"url": "https://example.com", "title": "Hello World"},
			},
			params: map[string]any{
				"entity": "{{event.id}}",
				"link":   "{{event.url}}",
				"title":  "{{event.title}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "entity-123", rendered["entity"])
				assert.Equal(t, "https://example.com", rendered["link"])
				assert.Equal(t, "Hello World", rendered["title"])
			},
		},
		{
			name:  "no-templates",
			event: types.DataEvent{},
			params: map[string]any{
				"key": "value",
				"num": 42,
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "value", rendered["key"])
				assert.Equal(t, 42, rendered["num"])
			},
		},
		{
			name: "step-references",
			recordSteps: map[string]map[string]any{
				"archive": {"id": "archive-1", "url": "https://archived.example.com"},
			},
			params: map[string]any{
				"ref_id":  "{{steps.archive.id}}",
				"ref_url": "{{steps.archive.url}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "archive-1", rendered["ref_id"])
				assert.Equal(t, "https://archived.example.com", rendered["ref_url"])
			},
		},
		{
			name: "nested-map",
			event: types.DataEvent{
				EventID:  "evt1",
				EntityID: "123",
			},
			params: map[string]any{
				"nested": map[string]any{
					"inner": "{{event.id}}",
				},
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				nested := rendered["nested"].(map[string]any)
				assert.Equal(t, "123", nested["inner"])
			},
		},
		{
			name: "string-slice",
			event: types.DataEvent{
				EventID:  "evt1",
				EntityID: "eid",
			},
			params: map[string]any{
				"items": []any{"{{event.id}}", "static"},
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				items := rendered["items"].([]any)
				assert.Equal(t, "eid", items[0])
				assert.Equal(t, "static", items[1])
			},
		},
		{
			name: "missing-event-field",
			params: map[string]any{
				"ref": "{{event.url}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Empty(t, rendered["ref"])
			},
		},
		{
			name: "non-string-event-field",
			event: types.DataEvent{
				Data: types.KV{"url": 42},
			},
			params: map[string]any{
				"ref": "{{event.url}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "42", rendered["ref"])
			},
		},
		{
			name: "json-field",
			event: types.DataEvent{
				Data: types.KV{"url": map[string]any{"href": "https://x.com"}},
			},
			params: map[string]any{
				"ref": "{{json (event \"url\")}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.JSONEq(t, `{"href":"https://x.com"}`, rendered["ref"].(string))
			},
		},
		{
			name: "condition",
			event: types.DataEvent{
				Data: types.KV{"status": "done"},
			},
			params: map[string]any{
				"action": "{{if eq .Event.status \"done\"}}archive{{else}}skip{{end}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "archive", rendered["action"])
			},
		},
		{
			name: "condition-else",
			event: types.DataEvent{
				Data: types.KV{"status": "pending"},
			},
			params: map[string]any{
				"action": "{{if eq .Event.status \"done\"}}archive{{else}}skip{{end}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "skip", rendered["action"])
			},
		},
		{
			name: "loop",
			event: types.DataEvent{
				Data: types.KV{"tags": []any{"a", "b", "c"}},
			},
			params: map[string]any{
				"joined": "{{range .Event.tags}}{{.}}-{{end}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "a-b-c-", rendered["joined"])
			},
		},
		{
			name: "loop-else",
			event: types.DataEvent{
				Data: types.KV{"tags": []any{}},
			},
			params: map[string]any{
				"result": "{{range .Event.tags}}x{{else}}empty{{end}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "empty", rendered["result"])
			},
		},
		{
			name: "nested-condition-loop",
			event: types.DataEvent{
				Data: types.KV{"items": []any{"a", "", "c"}},
			},
			params: map[string]any{
				"filtered": "{{range .Event.items}}{{if .}}{{.}},{{end}}{{end}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "a,c,", rendered["filtered"])
			},
		},
		{
			name: "default",
			event: types.DataEvent{
				Data: types.KV{},
			},
			params: map[string]any{
				"label": "{{default \"unknown\" .Event.category}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "unknown", rendered["label"])
			},
		},
		{
			name: "join-and-contains",
			event: types.DataEvent{
				Data: types.KV{"tags": []any{"alpha", "beta", "gamma"}},
			},
			params: map[string]any{
				"csv": "{{join .Event.tags \",\"}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "alpha,beta,gamma", rendered["csv"])
			},
		},
		{
			name:      "invalid-template",
			params:    map[string]any{"bad": "{{if xxx}}"},
			assertErr: "template:",
		},
		{
			name: "event-top-level-fields",
			event: types.DataEvent{
				EventID:   "evt-001",
				EventType: "bookmark.created",
				EntityID:  "entity-123",
				Source:    "test",
			},
			params: map[string]any{
				"event_id":   "{{event.event_id}}",
				"event_type": "{{event.event_type}}",
				"id":         "{{event.id}}",
				"entity_id":  "{{event.entity_id}}",
				"source":     "{{event.source}}",
			},
			asserts: func(t *testing.T, rendered map[string]any) {
				assert.Equal(t, "evt-001", rendered["event_id"])
				assert.Equal(t, "bookmark.created", rendered["event_type"])
				assert.Equal(t, "entity-123", rendered["id"])
				assert.Equal(t, "entity-123", rendered["entity_id"])
				assert.Equal(t, "test", rendered["source"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := NewRenderContext(tt.event)
			for name, data := range tt.recordSteps {
				rc.RecordStepResult(name, data)
			}
			rendered, err := rc.RenderParams(tt.params)
			if tt.assertErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.assertErr)
				return
			}
			require.NoError(t, err)
			tt.asserts(t, rendered)
		})
	}
}

func TestRenderString(t *testing.T) {
	tests := []struct {
		name     string
		event    types.DataEvent
		template string
		expected string
	}{
		{
			name: "render-string",
			event: types.DataEvent{
				EntityID: "123",
				Data:     types.KV{"url": "https://example.com"},
			},
			template: "id={{event.id}} url={{event.url}}",
			expected: "id=123 url=https://example.com",
		},
		{
			name: "render-string-new-syntax",
			event: types.DataEvent{
				EntityID: "123",
			},
			template: "{{if .Event.id}}ID:{{.Event.id}}{{end}}",
			expected: "ID:123",
		},
		{
			name: "event-top-level-fields",
			event: types.DataEvent{
				EventID:   "evt-001",
				EventType: "bookmark.created",
				EntityID:  "entity-123",
				Source:    "test",
			},
			template: "{{event.event_id}}:{{event.entity_id}}:{{event.source}}",
			expected: "evt-001:entity-123:test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := NewRenderContext(tt.event)
			result, err := rc.RenderString(tt.template)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildBackoff(t *testing.T) {
	tests := []struct {
		name string
		cfg  *types.RetryConfig
		test func(t *testing.T, bo backoff.BackOff)
	}{
		{
			name: "no-config",
			cfg:  &types.RetryConfig{},
			test: func(t *testing.T, bo backoff.BackOff) {
				require.NotNil(t, bo)
			},
		},
		{
			name: "exponential",
			cfg: &types.RetryConfig{
				MaxAttempts: 3,
				Delay:       1 * time.Second,
				Backoff:     types.BackoffExponential,
				MaxDelay:    10 * time.Second,
			},
			test: func(t *testing.T, bo backoff.BackOff) {
				require.NotNil(t, bo)
				assert.NotEqual(t, backoff.Stop, bo.NextBackOff())
			},
		},
		{
			name: "fixed",
			cfg: &types.RetryConfig{
				MaxAttempts: 3,
				Delay:       500 * time.Millisecond,
				Backoff:     types.BackoffFixed,
			},
			test: func(t *testing.T, bo backoff.BackOff) {
				require.NotNil(t, bo)
				delay := bo.NextBackOff()
				assert.Equal(t, 500*time.Millisecond, delay)
			},
		},
		{
			name: "linear",
			cfg: &types.RetryConfig{
				MaxAttempts: 3,
				Delay:       1 * time.Second,
				Backoff:     types.BackoffLinear,
				MaxDelay:    30 * time.Second,
			},
			test: func(t *testing.T, bo backoff.BackOff) {
				require.NotNil(t, bo)
				assert.NotEqual(t, backoff.Stop, bo.NextBackOff())
			},
		},
		{
			name: "with-jitter",
			cfg: &types.RetryConfig{
				MaxAttempts: 3,
				Delay:       1 * time.Second,
				Backoff:     types.BackoffExponential,
				Jitter:      true,
			},
			test: func(t *testing.T, bo backoff.BackOff) {
				require.NotNil(t, bo)
				assert.NotEqual(t, backoff.Stop, bo.NextBackOff())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bo := tt.cfg.BuildBackOff()
			tt.test(t, bo)
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		cfg  *types.RetryConfig
		want bool
	}{
		{
			name: "no-filter",
			err:  fmt.Errorf("generic error"),
			cfg:  &types.RetryConfig{},
			want: true,
		},
		{
			name: "nil-config",
			err:  fmt.Errorf("generic error"),
			cfg:  nil,
			want: true,
		},
		{
			name: "retry-on-match",
			err:  &types.Error{Code: "timeout", Retryable: true},
			cfg:  &types.RetryConfig{RetryOn: []string{"timeout", "rate_limited"}},
			want: true,
		},
		{
			name: "retry-on-no-match",
			err:  fmt.Errorf("some other error"),
			cfg:  &types.RetryConfig{RetryOn: []string{"timeout"}},
			want: false,
		},
		{
			name: "retryable-flag",
			err:  &types.Error{Retryable: true},
			cfg:  &types.RetryConfig{RetryOn: []string{"timeout"}},
			want: true,
		},
		{
			name: "retry-enabled",
			err:  nil,
			cfg:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "retry-enabled" {
				assert.False(t, (*types.RetryConfig)(nil).RetryEnabled())
				assert.False(t, (&types.RetryConfig{}).RetryEnabled())
				assert.True(t, (&types.RetryConfig{MaxAttempts: 3}).RetryEnabled())
				return
			}
			got := isRetryable(tt.err, tt.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertToTypesKV(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]any
		test func(t *testing.T, result types.KV)
	}{
		{
			name: "normal",
			in:   map[string]any{"a": 1, "b": "x"},
			test: func(t *testing.T, result types.KV) {
				assert.Equal(t, 1, result["a"])
				assert.Equal(t, "x", result["b"])
				assert.Len(t, result, 2)
			},
		},
		{
			name: "empty",
			in:   map[string]any{},
			test: func(t *testing.T, result types.KV) {
				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToTypesKV(tt.in)
			tt.test(t, result)
		})
	}
}

func TestNewEngine(t *testing.T) {
	e := NewEngine(nil, nil)
	assert.NotNil(t, e)
	assert.NotNil(t, e.Handler())
}

func TestCheckpointDataMarshaling(t *testing.T) {
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

func FuzzIsRetryable(f *testing.F) {
	f.Add("generic error", "timeout")
	f.Add("timeout error", "")
	f.Add("some error", "rate_limited")

	f.Fuzz(func(t *testing.T, errMsg, retryCode string) {
		err := errors.New(errMsg)
		cfg := &types.RetryConfig{}
		if retryCode != "" {
			cfg.RetryOn = []string{retryCode}
		}
		_ = isRetryable(err, cfg)
	})
}

func FuzzIsRetryableTypedError(f *testing.F) {
	f.Add("ERR001", "ERR001")
	f.Add("ERR001", "ERR002")

	f.Fuzz(func(t *testing.T, code, filter string) {
		err := &types.Error{Code: code, Retryable: false}
		cfg := &types.RetryConfig{RetryOn: []string{filter}}
		_ = isRetryable(err, cfg)
	})
}

func FuzzContainsErrorCode(f *testing.F) {
	f.Add("ERR001", "ERR001")
	f.Add("some error", "timeout")

	f.Fuzz(func(t *testing.T, code, target string) {
		err := &types.Error{Code: code}
		wrapped := fmt.Errorf("wrapped: %w", err)
		_ = containsErrorCode(wrapped, target)
		_ = containsErrorCode(err, target)
	})
}

func FuzzExtractResult(f *testing.F) {
	f.Add([]byte(`{"key":"value"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[1,2,3]`))
	f.Add([]byte(`"string"`))
	f.Add([]byte(`42`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var val any
		if err := sonic.Unmarshal(data, &val); err != nil {
			t.Skip()
		}
		res := &ability.InvokeResult{Data: val}
		result := extractResult(res)
		if result == nil {
			t.Error("extractResult returned nil")
		}
	})
}

func FuzzFindByEvent(f *testing.F) {
	f.Add([]byte(`[{"n":"p1","e":"e1"},{"n":"p2","e":"e2"}]`), "e1")
	f.Add([]byte(`[]`), "e1")

	f.Fuzz(func(t *testing.T, defsData []byte, eventType string) {
		var raw []struct {
			Name  string `json:"n"`
			Event string `json:"e"`
		}
		if err := sonic.Unmarshal(defsData, &raw); err != nil {
			t.Skip()
		}
		defs := make([]Definition, len(raw))
		for i, r := range raw {
			defs[i] = Definition{
				Name:    r.Name,
				Trigger: Trigger{Event: r.Event},
			}
		}
		result := FindByEvent(defs, eventType)
		// Each matched definition should have the trigger event matching eventType
		for _, d := range result {
			if d.Trigger.Event != eventType {
				t.Errorf("FindByEvent matched definition %q with event %q, expected %q",
					d.Name, d.Trigger.Event, eventType)
			}
		}
	})
}

func FuzzBuildStepResults(f *testing.F) {
	f.Add([]byte(`{"s1":{"id":"123"}}`))
	f.Add([]byte(`{}`))

	f.Fuzz(func(t *testing.T, stepsData []byte) {
		var raw map[string]map[string]any
		if err := sonic.Unmarshal(stepsData, &raw); err != nil {
			t.Skip()
		}
		event := types.DataEvent{EventID: "evt", EntityID: "123"}
		rc := NewRenderContext(event)
		for name, data := range raw {
			rc.RecordStepResult(name, data)
		}
		results := buildStepResults(rc)
		if len(results) != len(raw) {
			t.Errorf("buildStepResults len=%d, want %d", len(results), len(raw))
		}
		for name, sr := range results {
			if sr.Name != name {
				t.Errorf("StepResult name mismatch: %q != %q", sr.Name, name)
			}
		}
	})
}

func FuzzConvertRetryConfig(f *testing.F) {
	f.Add(0, "1s", "", "exponential", "")
	f.Add(3, "500ms", "10s", "fixed", "")
	f.Add(0, "", "", "", "")

	f.Fuzz(func(t *testing.T, maxAttempts int, delay, maxDelay, backoffStr, jitterStr string) {
		cfg := &config.PipelineStepRetry{
			MaxAttempts: maxAttempts,
			Delay:       delay,
			MaxDelay:    maxDelay,
			Backoff:     backoffStr,
		}
		_ = jitterStr
		result, err := convertRetryConfig(cfg)
		_ = result
		_ = err
	})
}

func FuzzErrorWrapping(f *testing.F) {
	f.Add("outer", "ERR001")
	f.Add("outer", "")

	f.Fuzz(func(t *testing.T, outer, code string) {
		inner := &types.Error{Code: code, Retryable: true}
		wrapped := fmt.Errorf("%s: %w", outer, inner)
		if errors.As(wrapped, new(*types.Error)) {
			_ = containsErrorCode(wrapped, code)
		}
		_ = isRetryable(wrapped, &types.RetryConfig{RetryOn: []string{code}})
	})
}
