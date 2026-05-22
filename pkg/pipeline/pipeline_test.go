package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/types"
)

var (
	noopPC = metrics.NewPipelineCollector(nil)
	noopEC = metrics.NewEventCollector(nil)
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			defs := LoadConfig(tt.cfg)
			tt.test(t, defs)
		})
	}
}

func TestLoadConfig_CronTrigger(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cfg     []config.Pipeline
		asserts func(t *testing.T, defs []Definition)
	}{
		{
			name: "cron only definition",
			cfg: []config.Pipeline{
				{
					Name:    "cron-pl",
					Enabled: true,
					Trigger: config.PipelineTrigger{Cron: "0 */6 * * *"},
				},
			},
			asserts: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Equal(t, "0 */6 * * *", defs[0].Trigger.Cron)
				assert.Empty(t, defs[0].Trigger.Event)
			},
		},
		{
			name: "both event and cron",
			cfg: []config.Pipeline{
				{
					Name:    "mixed-pl",
					Enabled: true,
					Trigger: config.PipelineTrigger{Event: "e1", Cron: "@daily"},
				},
			},
			asserts: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Equal(t, "e1", defs[0].Trigger.Event)
				assert.Equal(t, "@daily", defs[0].Trigger.Cron)
			},
		},
		{
			name: "invalid cron expression skipped",
			cfg: []config.Pipeline{
				{
					Name:    "bad-cron",
					Enabled: true,
					Trigger: config.PipelineTrigger{Cron: "not-a-valid-cron"},
				},
				{
					Name:    "good-cron",
					Enabled: true,
					Trigger: config.PipelineTrigger{Cron: "0 0 * * *"},
				},
			},
			asserts: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Equal(t, "good-cron", defs[0].Name)
			},
		},
		{
			name: "disabled pipeline not loaded",
			cfg: []config.Pipeline{
				{
					Name:    "disabled-pl",
					Enabled: false,
					Trigger: config.PipelineTrigger{Cron: "0 0 * * *"},
				},
			},
			asserts: func(t *testing.T, defs []Definition) {
				assert.Empty(t, defs)
			},
		},
		{
			name: "cron with timeout",
			cfg: []config.Pipeline{
				{
					Name:    "timeout-pl",
					Enabled: true,
					Trigger: config.PipelineTrigger{Cron: "0 0 * * *", CronTimeout: "30m"},
				},
			},
			asserts: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Equal(t, 30*time.Minute, defs[0].Trigger.CronTimeout)
			},
		},
		{
			name: "invalid cron_timeout still loads pipeline",
			cfg: []config.Pipeline{
				{
					Name:    "bad-timeout",
					Enabled: true,
					Trigger: config.PipelineTrigger{Cron: "0 0 * * *", CronTimeout: "not-a-duration"},
				},
			},
			asserts: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Equal(t, "bad-timeout", defs[0].Name)
				assert.Equal(t, time.Duration(0), defs[0].Trigger.CronTimeout)
			},
		},
		{
			name: "event-only pipeline gets default timeout",
			cfg: []config.Pipeline{
				{
					Name:    "event-only",
					Enabled: true,
					Trigger: config.PipelineTrigger{Event: "e1"},
				},
			},
			asserts: func(t *testing.T, defs []Definition) {
				require.Len(t, defs, 1)
				assert.Equal(t, 10*time.Minute, defs[0].Trigger.CronTimeout)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defs := LoadConfig(tt.cfg)
			tt.asserts(t, defs)
		})
	}
}

func TestFindByEvent(t *testing.T) {
	t.Parallel()
	t.Run("method-match", func(t *testing.T) {
		t.Parallel()
		d := Definition{Name: "p", Trigger: Trigger{Event: "bookmark.created"}}
		matched := d.FindByEvent("bookmark.created")
		require.Len(t, matched, 1)
		assert.Equal(t, "p", matched[0].Name)
	})

	t.Run("method-no-match", func(t *testing.T) {
		t.Parallel()
		d := Definition{Name: "p", Trigger: Trigger{Event: "bookmark.created"}}
		matched := d.FindByEvent("archive.created")
		assert.Empty(t, matched)
	})

	t.Run("package-multiple-matches", func(t *testing.T) {
		t.Parallel()
		defs := []Definition{
			{Name: "p1", Trigger: Trigger{Event: "e1"}},
			{Name: "p2", Trigger: Trigger{Event: "e2"}},
			{Name: "p3", Trigger: Trigger{Event: "e1"}},
		}
		matched := FindByEvent(defs, "e1")
		require.Len(t, matched, 2)
	})

	t.Run("package-no-matches", func(t *testing.T) {
		t.Parallel()
		defs := []Definition{
			{Name: "p1", Trigger: Trigger{Event: "e1"}},
		}
		matched := FindByEvent(defs, "nonexistent")
		assert.Empty(t, matched)
	})

	t.Run("package-empty-slice", func(t *testing.T) {
		t.Parallel()
		matched := FindByEvent(nil, "e1")
		assert.Empty(t, matched)
	})
}

func TestRenderContext(t *testing.T) {
	t.Parallel()
	t.Run("new-render-context", func(t *testing.T) {
		t.Parallel()
		event := types.DataEvent{EventID: "evt1", EventType: "bookmark.created", EntityID: "123"}
		rc := NewRenderContext(event)
		assert.Equal(t, "evt1", rc.Event.EventID)
		assert.NotNil(t, rc.Steps)
		assert.Empty(t, rc.Steps)
	})

	t.Run("record-step-result", func(t *testing.T) {
		t.Parallel()
		rc := NewRenderContext(types.DataEvent{})
		rc.RecordStepResult("step1", map[string]any{"id": "abc", "url": "https://x.com"})
		assert.Contains(t, rc.Steps, "step1")
		assert.Equal(t, "abc", rc.Steps["step1"]["id"])
	})

	t.Run("build-step-results", func(t *testing.T) {
		t.Parallel()
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
	t.Parallel()
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
				nested, ok := rendered["nested"].(map[string]any)
				assert.True(t, ok)
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
				items, ok := rendered["items"].([]any)
				assert.True(t, ok)
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
			t.Parallel()
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
	t.Parallel()
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
			t.Parallel()
			rc := NewRenderContext(tt.event)
			result, err := rc.RenderString(tt.template)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertToTypesKV(t *testing.T) {
	t.Parallel()
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
		{
			name: "nil map",
			in:   nil,
			test: func(t *testing.T, result types.KV) {
				assert.NotNil(t, result)
				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertToTypesKV(tt.in)
			tt.test(t, result)
		})
	}
}

func TestNewEngine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		defs  []Definition
		store RunStore
	}{
		{
			name:  "nil-params",
			defs:  nil,
			store: nil,
		},
		{
			name:  "empty-defs",
			defs:  []Definition{},
			store: nil,
		},
		{
			name: "with-defs",
			defs: []Definition{
				{Name: "p1", Trigger: Trigger{Event: "e1"}},
			},
			store: nil,
		},
		{
			name: "mixed-enabled-defs",
			defs: []Definition{
				{Name: "enabled", Enabled: true},
				{Name: "disabled", Enabled: false},
			},
			store: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(tt.defs, tt.store, nil, noopPC, noopEC)
			assert.NotNil(t, e)
			assert.NotNil(t, e.Handler())
		})
	}
}

func TestCheckpointDataMarshaling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		cp    *CheckpointData
		check func(t *testing.T, restored CheckpointData)
	}{
		{
			name: "basic",
			cp: &CheckpointData{
				StepIndex: 2,
				StepResults: map[string]*StepResult{
					"step1": {Name: "step1", Output: map[string]any{"id": "123"}},
				},
				HeartbeatAt: time.Now(),
			},
			check: func(t *testing.T, restored CheckpointData) {
				assert.Equal(t, 2, restored.StepIndex)
				assert.Equal(t, "step1", restored.StepResults["step1"].Name)
				assert.NotNil(t, restored.StepResults["step1"].Output)
				assert.Equal(t, "123", restored.StepResults["step1"].Output["id"])
			},
		},
		{
			name: "empty-step-results",
			cp: &CheckpointData{
				StepIndex:   0,
				HeartbeatAt: time.Now(),
			},
			check: func(t *testing.T, restored CheckpointData) {
				assert.Equal(t, 0, restored.StepIndex)
				assert.Empty(t, restored.StepResults)
			},
		},
		{
			name: "multiple-steps",
			cp: &CheckpointData{
				StepIndex: 5,
				StepResults: map[string]*StepResult{
					"step1": {Name: "step1", Output: map[string]any{"a": "1"}},
					"step2": {Name: "step2", Output: map[string]any{"b": "2"}},
				},
				HeartbeatAt: time.Now(),
			},
			check: func(t *testing.T, restored CheckpointData) {
				assert.Equal(t, 5, restored.StepIndex)
				assert.Len(t, restored.StepResults, 2)
				assert.Equal(t, "step1", restored.StepResults["step1"].Name)
				assert.Equal(t, "step2", restored.StepResults["step2"].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := sonic.Marshal(tt.cp)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			var restored CheckpointData
			err = sonic.Unmarshal(data, &restored)
			require.NoError(t, err)
			tt.check(t, restored)
		})
	}
}

type mockAuditor struct {
	entries []audit.Entry
}

func (m *mockAuditor) Record(_ context.Context, entry audit.Entry) error {
	m.entries = append(m.entries, entry)
	return nil
}
func (m *mockAuditor) RecordSuccess(_ context.Context, entry audit.Entry) error {
	return m.Record(context.TODO(), entry)
}
func (m *mockAuditor) RecordFailure(_ context.Context, entry audit.Entry, _ error) error {
	return m.Record(context.TODO(), entry)
}
func (m *mockAuditor) RecordRejected(_ context.Context, entry audit.Entry, _ string) error {
	return m.Record(context.TODO(), entry)
}

func TestEngine_Audit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		pipelineName  string
		event         types.DataEvent
		expectActions []string
	}{
		{
			name:          "audit start and complete for no-step pipeline",
			pipelineName:  "audit-pl",
			event:         types.DataEvent{EventID: "evt1", EventType: "test.event"},
			expectActions: []string{"pipeline.start", "pipeline.complete"},
		},
		{
			name:          "empty event with empty pipeline name",
			pipelineName:  "",
			event:         types.DataEvent{EventID: "", EventType: "test.event"},
			expectActions: []string{"pipeline.start", "pipeline.complete"},
		},
		{
			name:          "non-empty pipeline with empty event",
			pipelineName:  "pl1",
			event:         types.DataEvent{EventType: "test.event"},
			expectActions: []string{"pipeline.start", "pipeline.complete"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &mockAuditor{}
			defs := []Definition{
				{
					Name:    tt.pipelineName,
					Enabled: true,
					Trigger: Trigger{Event: "test.event"},
					Steps:   []Step{},
				},
			}
			e := NewEngine(defs, nil, m, noopPC, noopEC)
			_ = e.Handler()(context.Background(), tt.event)
			require.Len(t, m.entries, len(tt.expectActions))
			for i, expected := range tt.expectActions {
				assert.Equal(t, expected, m.entries[i].Action)
				assert.Equal(t, "pipeline", m.entries[i].Target.Type)
				assert.Equal(t, tt.pipelineName, m.entries[i].Target.ID)
			}
		})
	}
}

func TestNewEngine_WithAuditor(t *testing.T) {
	t.Parallel()
	m := &mockAuditor{}
	tests := []struct {
		name    string
		auditor audit.Auditor
	}{
		{name: "with mock auditor", auditor: m},
		{name: "with nil auditor", auditor: nil},
		{name: "with nil interface", auditor: audit.Auditor(nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(nil, nil, tt.auditor, noopPC, noopEC)
			assert.NotNil(t, e)
			assert.NotNil(t, e.Handler())
		})
	}
}
