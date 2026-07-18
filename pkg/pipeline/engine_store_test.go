package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/types"
)

type mockPipelineStore struct {
	mu                   sync.Mutex
	runs                 map[int64]*gen.PipelineRun
	stepRuns             map[int64]*gen.PipelineStepRun
	checkpoints          map[int64]*CheckpointData
	consumed             map[string]map[string]bool
	links                []*gen.ResourceLink
	nextRunID            int64
	nextStepID           int64
	heartbeats           int
	createRunErr         error
	hasConsumed          bool
	hasConsumedErr       error
	recordConsumptionErr error
}

func newMockPipelineStore() *mockPipelineStore {
	return &mockPipelineStore{
		runs:        make(map[int64]*gen.PipelineRun),
		stepRuns:    make(map[int64]*gen.PipelineStepRun),
		checkpoints: make(map[int64]*CheckpointData),
		consumed:    make(map[string]map[string]bool),
	}
}

func (m *mockPipelineStore) CreateRun(_ context.Context, pipelineName, eventID, eventType, triggerSource string) (*gen.PipelineRun, error) {
	if m.createRunErr != nil {
		return nil, m.createRunErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextRunID++
	run := &gen.PipelineRun{
		ID:            m.nextRunID,
		PipelineName:  pipelineName,
		EventID:       eventID,
		EventType:     eventType,
		TriggerSource: pipelinerun.TriggerSource(triggerSource),
		Status:        int(schema.PipelineStart),
	}
	m.runs[run.ID] = run
	return run, nil
}

func (m *mockPipelineStore) UpdateRunStatus(_ context.Context, runID int64, status int, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if run, ok := m.runs[runID]; ok {
		run.Status = status
		run.Error = errMsg
	}
	return nil
}

func (m *mockPipelineStore) CreateStepRun(_ context.Context, runID int64, stepName, capName, operation string, params map[string]any, attempt int) (*gen.PipelineStepRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextStepID++
	sr := &gen.PipelineStepRun{
		ID:            m.nextStepID,
		PipelineRunID: runID,
		StepName:      stepName,
		Capability:    capName,
		Operation:     operation,
		Params:        params,
		Attempt:       attempt,
		Status:        int(schema.PipelineStart),
	}
	m.stepRuns[sr.ID] = sr
	return sr, nil
}

func (m *mockPipelineStore) UpdateStepRun(_ context.Context, stepRunID int64, status int, result map[string]any, errMsg string, attempt int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sr, ok := m.stepRuns[stepRunID]; ok {
		sr.Status = status
		sr.Result = result
		sr.Error = errMsg
		sr.Attempt = attempt
	}
	return nil
}

func (m *mockPipelineStore) SaveCheckpoint(_ context.Context, runID int64, data any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp, ok := data.(*CheckpointData)
	if !ok {
		return nil
	}
	cpCopy := *cp
	m.checkpoints[runID] = &cpCopy
	return nil
}

func (m *mockPipelineStore) GetIncompleteRuns(context.Context) ([]*gen.PipelineRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil, nil
}

func (m *mockPipelineStore) GetCheckpoint(_ context.Context, runID int64, target any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp, ok := m.checkpoints[runID]
	if !ok {
		return errors.New("checkpoint not found")
	}
	dest, ok := target.(*CheckpointData)
	if !ok {
		return errors.New("invalid checkpoint target")
	}
	*dest = *cp
	return nil
}

func (m *mockPipelineStore) GetRun(_ context.Context, runID int64) (*gen.PipelineRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.runs[runID]
	if !ok {
		return nil, errors.New("run not found")
	}
	return run, nil
}

func (m *mockPipelineStore) UpdateRunHeartbeat(context.Context, int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.heartbeats++
	return nil
}

func (m *mockPipelineStore) HasConsumed(_ context.Context, consumerName, eventID string) (bool, error) {
	if m.hasConsumedErr != nil {
		return false, m.hasConsumedErr
	}
	if m.hasConsumed {
		return true, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if byConsumer, ok := m.consumed[consumerName]; ok {
		return byConsumer[eventID], nil
	}
	return false, nil
}

func (m *mockPipelineStore) RecordConsumption(_ context.Context, consumerName, eventID string) error {
	if m.recordConsumptionErr != nil {
		return m.recordConsumptionErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.consumed[consumerName] == nil {
		m.consumed[consumerName] = make(map[string]bool)
	}
	m.consumed[consumerName][eventID] = true
	return nil
}

func (m *mockPipelineStore) RecordResourceLink(_ context.Context, link *gen.ResourceLink) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.links = append(m.links, link)
	return nil
}

func registerExampleInvoker(t *testing.T, operation string, fn capability.Invoker) {
	t.Helper()
	require.NoError(t, capability.RegisterInvoker(hub.CapExample, operation, fn))
	t.Cleanup(func() {
		capability.UnregisterInvoker(hub.CapExample, operation)
	})
}

func TestFormatStepError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		step    string
		err     error
		attempt int
		want    string
	}{
		{
			name:    "default single attempt",
			step:    "fetch",
			err:     errors.New("boom"),
			attempt: 1,
			want:    "step fetch: boom",
		},
		{
			name:    "retries exhausted",
			step:    "fetch",
			err:     errors.New("boom"),
			attempt: 3,
			want:    "step fetch (retries exhausted): boom",
		},
		{
			name:    "context canceled",
			step:    "fetch",
			err:     context.Canceled,
			attempt: 1,
			want:    "step fetch cancelled: context canceled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := formatStepError(tt.step, tt.err, tt.attempt)
			require.Error(t, err)
			assert.Equal(t, tt.want, err.Error())
		})
	}
}

func TestTriggerDescription(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		trigger Trigger
		want    string
	}{
		{name: "event trigger", trigger: Trigger{Event: "bookmark.created"}, want: "event:bookmark.created"},
		{name: "webhook trigger", trigger: Trigger{Webhook: &WebhookConfig{Path: "hooks/gh"}}, want: "webhook:hooks/gh"},
		{name: "cron trigger", trigger: Trigger{Cron: "@daily"}, want: "cron:@daily"},
		{name: "unknown trigger", trigger: Trigger{}, want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, triggerDescription(tt.trigger))
		})
	}
}

func TestEngine_CheckDedupAndRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		newStore func() RunStore
		preload  func(RunStore)
		eventID  string
		wantDone bool
		wantErr  bool
	}{
		{
			name:     "nil store skips dedup",
			newStore: func() RunStore { return nil },
			eventID:  "e1",
			wantDone: false,
		},
		{
			name:     "first event is recorded",
			newStore: func() RunStore { return newMockPipelineStore() },
			eventID:  "e1",
			wantDone: false,
		},
		{
			name:     "already consumed event skipped",
			newStore: func() RunStore { return newMockPipelineStore() },
			preload: func(s RunStore) {
				ms, ok := s.(*mockPipelineStore)
				require.True(t, ok)
				ms.hasConsumed = true
			},
			eventID:  "e2",
			wantDone: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := tt.newStore()
			if tt.preload != nil {
				tt.preload(store)
			}
			e := NewEngine(nil, store, nil, noopPC, metrics.NewEventCollector(nil))
			defer e.Stop()
			done, err := e.checkDedupAndRecord(context.Background(), "pl", tt.eventID, "evt.type")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDone, done)
		})
	}
}

func TestEngine_ExecutePipelineWithStore(t *testing.T) {
	t.Parallel()
	registerExampleInvoker(t, "echo", func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{
			Capability: hub.CapExample,
			Operation:  "echo",
			Data:       map[string]any{"value": params["value"]},
		}, nil
	})

	store := newMockPipelineStore()
	pc := metrics.NewPipelineCollector(nil)
	ec := metrics.NewEventCollector(nil)
	cb := &mockStepCallback{}
	def := Definition{
		Name:      "stored-pl",
		Enabled:   true,
		Resumable: true,
		Trigger:   Trigger{Event: "item.created"},
		Steps: []Step{
			{Name: "s1", Capability: hub.CapExample, Operation: "echo", Params: map[string]any{"value": "hello"}},
		},
	}
	e := NewEngine([]Definition{def}, store, nil, pc, ec)
	defer e.Stop()
	e.SetCallback(cb)

	event := types.DataEvent{EventID: "evt-1", EventType: "item.created", EntityID: "src-1", App: "app-a"}
	err := e.executePipeline(context.Background(), def, event, "event")
	require.NoError(t, err)
	assert.NotEmpty(t, store.runs)
	assert.NotEmpty(t, store.stepRuns)
	assert.NotEmpty(t, store.checkpoints)
	assert.NotEmpty(t, cb.calls)
}

func TestEngine_SaveResourceLink(t *testing.T) {
	t.Parallel()
	registerExampleInvoker(t, "create", func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{
			Capability: hub.CapExample,
			Operation:  "create",
			Data:       map[string]any{"id": "new-1"},
			Resource: &capability.ResourceMeta{
				EventID:  "target-ev",
				EntityID: "ent-2",
				App:      "target-app",
			},
		}, nil
	})

	store := newMockPipelineStore()
	def := Definition{
		Name:    "link-pl",
		Enabled: true,
		Trigger: Trigger{Event: "item.created"},
		Steps: []Step{
			{Name: "create-item", Capability: hub.CapExample, Operation: "create"},
		},
	}
	e := NewEngine([]Definition{def}, store, nil, noopPC, noopEC)
	defer e.Stop()

	event := types.DataEvent{
		EventID: "src-ev", EventType: "item.created", EntityID: "src-ent", App: "src-app",
		Capability: "source-cap",
	}
	err := e.executePipeline(context.Background(), def, event, "event")
	require.NoError(t, err)
	require.Len(t, store.links, 1)
	assert.Equal(t, "src-ev", store.links[0].SourceEventID)
	assert.Equal(t, "target-ev", store.links[0].TargetEventID)
}

func TestEngine_ResumePipeline(t *testing.T) {
	t.Parallel()
	registerExampleInvoker(t, "resume", func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: map[string]any{"ok": true}}, nil
	})

	store := newMockPipelineStore()
	run, err := store.CreateRun(context.Background(), "resume-pl", "evt-r", "item.updated", "event")
	require.NoError(t, err)
	event := types.DataEvent{EventID: "evt-r", EventType: "item.updated", EntityID: "1"}
	require.NoError(t, store.SaveCheckpoint(context.Background(), run.ID, &CheckpointData{
		StepIndex: 1,
		StepResults: map[string]*StepResult{
			"s1": {Name: "s1", Output: map[string]any{"seed": "v"}},
		},
		Event: event,
	}))

	def := Definition{
		Name:      "resume-pl",
		Enabled:   true,
		Resumable: true,
		Trigger:   Trigger{Event: "item.updated"},
		Steps: []Step{
			{Name: "s1", Capability: hub.CapExample, Operation: "resume"},
			{Name: "s2", Capability: hub.CapExample, Operation: "resume"},
		},
	}
	cb := &mockStepCallback{}
	e := NewEngine([]Definition{def}, store, nil, noopPC, noopEC)
	defer e.Stop()
	e.SetCallback(cb)

	err = e.ResumePipeline(context.Background(), run.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, cb.calls)
}

func TestEngine_ResumePipeline_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(t *testing.T) (*Engine, int64)
		errContains string
	}{
		{
			name: "missing store",
			setup: func(_ *testing.T) (*Engine, int64) {
				e := NewEngine(nil, nil, nil, noopPC, noopEC)
				return e, 1
			},
			errContains: "pipeline store not available",
		},
		{
			name: "invalid checkpoint index",
			setup: func(t *testing.T) (*Engine, int64) {
				store := newMockPipelineStore()
				run, err := store.CreateRun(context.Background(), "resume-pl", "evt", "t", "event")
				require.NoError(t, err)
				require.NoError(t, store.SaveCheckpoint(context.Background(), run.ID, &CheckpointData{StepIndex: -1}))
				def := Definition{Name: "resume-pl", Resumable: true, Trigger: Trigger{Event: "t"}}
				e := NewEngine([]Definition{def}, store, nil, noopPC, noopEC)
				return e, run.ID
			},
			errContains: "invalid checkpoint",
		},
		{
			name: "missing resumable definition",
			setup: func(t *testing.T) (*Engine, int64) {
				store := newMockPipelineStore()
				run, err := store.CreateRun(context.Background(), "gone-pl", "evt", "t", "event")
				require.NoError(t, err)
				require.NoError(t, store.SaveCheckpoint(context.Background(), run.ID, &CheckpointData{StepIndex: 0}))
				e := NewEngine([]Definition{{Name: "other", Resumable: true}}, store, nil, noopPC, noopEC)
				return e, run.ID
			},
			errContains: "no resumable pipeline definition",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e, runID := tt.setup(t)
			defer e.Stop()
			err := e.ResumePipeline(context.Background(), runID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestEngine_ExecuteCronJob(t *testing.T) {
	t.Parallel()
	registerExampleInvoker(t, "cron-op", func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{Data: map[string]any{"cron": true}}, nil
	})

	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)
	store := newMockPipelineStore()
	pc := metrics.NewPipelineCollector(nil)
	def := Definition{
		Name:    "cron-pl",
		Enabled: true,
		Trigger: Trigger{Cron: "@daily", CronTimeout: time.Minute},
		Steps:   []Step{{Name: "tick", Capability: hub.CapExample, Operation: "cron-op"}},
	}
	e := NewEngineWithClock([]Definition{def}, store, nil, pc, noopEC, clock)
	defer e.Stop()

	e.executeCronJob(context.Background(), def)
	assert.NotEmpty(t, store.runs)
}

func TestEngine_MutexFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		defs    []Definition
		lookup  string
		wantNil bool
	}{
		{name: "existing pipeline mutex", defs: []Definition{{Name: "p1"}}, lookup: "p1", wantNil: false},
		{name: "missing pipeline mutex", defs: []Definition{{Name: "p1"}}, lookup: "missing", wantNil: true},
		{name: "empty engine", defs: nil, lookup: "any", wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(tt.defs, nil, nil, noopPC, noopEC)
			defer e.Stop()
			mu := e.MutexFor(tt.lookup)
			if tt.wantNil {
				assert.Nil(t, mu)
				return
			}
			require.NotNil(t, mu)
		})
	}
}

func TestEngine_HeartbeatLoop(t *testing.T) {
	t.Parallel()
	store := newMockPipelineStore()
	e := NewEngine(nil, store, nil, noopPC, noopEC)
	defer e.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e.heartbeatLoop(ctx, 1, "pl")
	assert.Equal(t, 0, store.heartbeats)
}

func TestEngine_AuditPipelineEventWithStore(t *testing.T) {
	t.Parallel()
	auditor := &mockAuditor{}
	e := NewEngine(nil, nil, auditor, noopPC, noopEC)
	defer e.Stop()
	e.auditPipelineEvent(context.Background(), "pl", "pipeline.start", "e1", "evt")
	require.Len(t, auditor.entries, 1)
	assert.Equal(t, "pipeline.start", auditor.entries[0].Action)
}

func TestLoadConfig_RetryConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cfg       []config.Pipeline
		wantLen   int
		wantRetry bool
	}{
		{
			name: "valid exponential retry",
			cfg: []config.Pipeline{
				{
					Name: "retry-pl", Enabled: true,
					Steps: []config.PipelineStep{
						{
							Name: "s1", Capability: "example", Operation: "op",
							Retry: &config.PipelineStepRetry{
								MaxAttempts: 3,
								Delay:       "100ms",
								MaxDelay:    "1s",
								Backoff:     "exponential",
								Jitter:      true,
							},
						},
					},
				},
			},
			wantLen:   1,
			wantRetry: true,
		},
		{
			name: "invalid delay skips step",
			cfg: []config.Pipeline{
				{
					Name: "bad-retry", Enabled: true,
					Steps: []config.PipelineStep{
						{Name: "bad", Capability: "example", Operation: "op", Retry: &config.PipelineStepRetry{MaxAttempts: 2, Delay: "not-a-duration"}},
						{Name: "good", Capability: "example", Operation: "op"},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "fixed backoff multiplier",
			cfg: []config.Pipeline{
				{
					Name: "fixed-retry", Enabled: true,
					Steps: []config.PipelineStep{
						{
							Name: "s1", Capability: "example", Operation: "op",
							Retry: &config.PipelineStepRetry{MaxAttempts: 2, Delay: "50ms", Backoff: "fixed"},
						},
					},
				},
			},
			wantLen:   1,
			wantRetry: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defs := LoadConfig(tt.cfg)
			assert.Len(t, defs, tt.wantLen)
			if tt.wantRetry && len(defs) > 0 && len(defs[0].Steps) > 0 {
				require.NotNil(t, defs[0].Steps[0].Retry)
				assert.GreaterOrEqual(t, defs[0].Steps[0].Retry.MaxAttempts, 2)
			}
		})
	}
}

func TestLoadConfig_WebhookValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      []config.Pipeline
		wantDefs int
	}{
		{
			name: "invalid webhook method skipped",
			cfg: []config.Pipeline{
				{
					Name: "bad-method", Enabled: true,
					Trigger: config.PipelineTrigger{
						Webhook: &config.WebhookTrigger{Path: "p", Method: "DELETE", Auth: &config.WebhookAuth{Token: "t"}},
					},
				},
			},
			wantDefs: 0,
		},
		{
			name: "missing webhook auth skipped",
			cfg: []config.Pipeline{
				{
					Name: "no-auth", Enabled: true,
					Trigger: config.PipelineTrigger{Webhook: &config.WebhookTrigger{Path: "p"}},
				},
			},
			wantDefs: 0,
		},
		{
			name: "invalid payload mode skipped",
			cfg: []config.Pipeline{
				{
					Name: "bad-payload", Enabled: true,
					Trigger: config.PipelineTrigger{
						Webhook: &config.WebhookTrigger{
							Path: "p", Payload: "invalid", Auth: &config.WebhookAuth{Token: "t"},
						},
					},
				},
			},
			wantDefs: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defs := LoadConfig(tt.cfg)
			assert.Len(t, defs, tt.wantDefs)
		})
	}
}

type mockDefinitionReader struct {
	records []DefinitionRecord
	err     error
}

func (m *mockDefinitionReader) ListPublishedDefinitions(context.Context) ([]DefinitionRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.records, nil
}

func TestLoadFromDB(t *testing.T) {
	t.Parallel()
	yamlDef := `name: db-pl
enabled: true
resumable: true
triggers:
  - type: event
    enabled: true
    event: item.created
steps:
  - name: s1
    capability: example
    operation: list
`
	tests := []struct {
		name        string
		reader      DefinitionReader
		wantLen     int
		wantUID     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "nil reader returns nil",
			reader:  nil,
			wantLen: 0,
		},
		{
			name: "loads published definitions with creator uid",
			reader: &mockDefinitionReader{records: []DefinitionRecord{
				{Name: "db-pl", YAML: yamlDef, CreatedBy: "user-admin"},
			}},
			wantLen: 1,
			wantUID: "user-admin",
		},
		{
			name:        "reader error propagates",
			reader:      &mockDefinitionReader{err: errors.New("db down")},
			wantErr:     true,
			errContains: "load definitions from db",
		},
		{
			name: "invalid yaml returns error",
			reader: &mockDefinitionReader{records: []DefinitionRecord{
				{Name: "bad", YAML: "{{invalid"},
			}},
			wantErr:     true,
			errContains: "parse bad",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defs, err := LoadFromDB(context.Background(), tt.reader)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, defs, tt.wantLen)
			if tt.wantUID != "" {
				require.NotEmpty(t, defs)
				assert.Equal(t, tt.wantUID, defs[0].UID)
			}
		})
	}
}

func TestExtractResult(t *testing.T) {
	t.Parallel()
	res := &capability.InvokeResult{Data: map[string]any{"x": 1}}
	got := extractResult(res)
	assert.Equal(t, map[string]any{"x": 1}, got)
}

func TestEngine_HandleEventDedupSkipsExecution(t *testing.T) {
	t.Parallel()
	store := newMockPipelineStore()
	store.hasConsumed = true
	def := Definition{Name: "dedup-pl", Enabled: true, Trigger: Trigger{Event: "e1"}}
	e := NewEngine([]Definition{def}, store, nil, noopPC, metrics.NewEventCollector(nil))
	defer e.Stop()
	err := e.Handler()(context.Background(), types.DataEvent{EventID: "dup", EventType: "e1"})
	require.NoError(t, err)
	assert.Empty(t, store.runs)
}

func TestEngine_CreateRunRecordError(t *testing.T) {
	t.Parallel()
	store := newMockPipelineStore()
	store.createRunErr = errors.New("create failed")
	e := NewEngine(nil, store, nil, noopPC, noopEC)
	defer e.Stop()
	_, err := e.createRunRecord(context.Background(), "pl", "e1", "t", "event")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create run")
}

func TestEngine_FinishRunRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		failed     bool
		finalErr   error
		wantStatus int
	}{
		{
			name:       "success marks done",
			failed:     false,
			finalErr:   nil,
			wantStatus: int(schema.PipelineDone),
		},
		{
			name:       "business failure marks failed",
			failed:     true,
			finalErr:   errors.New("template test not found"),
			wantStatus: int(schema.PipelineFailed),
		},
		{
			name:       "context canceled marks cancelled",
			failed:     true,
			finalErr:   context.Canceled,
			wantStatus: int(schema.PipelineCancel),
		},
		{
			name:       "deadline exceeded marks cancelled",
			failed:     true,
			finalErr:   context.DeadlineExceeded,
			wantStatus: int(schema.PipelineCancel),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMockPipelineStore()
			e := NewEngine(nil, store, nil, noopPC, noopEC)
			defer e.Stop()
			run, err := store.CreateRun(context.Background(), "pl", "e1", "t", "event")
			require.NoError(t, err)
			e.finishRunRecord(context.Background(), run.ID, tt.failed, tt.finalErr)
			assert.Equal(t, tt.wantStatus, store.runs[run.ID].Status)
		})
	}
}

func TestTerminalStatusFromError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		err             error
		wantStatus      schema.PipelineState
		wantMetricLabel string
	}{
		{
			name:            "nil error is failed",
			err:             nil,
			wantStatus:      schema.PipelineFailed,
			wantMetricLabel: "failed",
		},
		{
			name:            "business error is failed",
			err:             errors.New("template test not found"),
			wantStatus:      schema.PipelineFailed,
			wantMetricLabel: "failed",
		},
		{
			name:            "context canceled is cancelled",
			err:             context.Canceled,
			wantStatus:      schema.PipelineCancel,
			wantMetricLabel: "cancel",
		},
		{
			name:            "wrapped deadline exceeded is cancelled",
			err:             fmt.Errorf("step message cancelled: %w", context.DeadlineExceeded),
			wantStatus:      schema.PipelineCancel,
			wantMetricLabel: "cancel",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotStatus, gotLabel := terminalStatusFromError(tt.err)
			assert.Equal(t, tt.wantStatus, gotStatus)
			assert.Equal(t, tt.wantMetricLabel, gotLabel)
		})
	}
}

func TestEngine_EmitRunStartComplete(t *testing.T) {
	t.Parallel()
	cb := &mockStepCallback{}
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)
	e := NewEngineWithClock(nil, nil, nil, noopPC, noopEC, clock)
	defer e.Stop()
	e.SetCallback(cb)

	def := &Definition{
		Name:    "emit-pl",
		Trigger: Trigger{Event: "e1"},
		Steps:   []Step{{Name: "only"}},
	}
	e.emitRunStart(context.Background(), 7, def)
	clock.Advance(time.Second)
	e.emitRunComplete(context.Background(), 7, def, seed, false, nil)
	require.Len(t, cb.calls, 2)
	assert.Equal(t, "OnRunStart", cb.calls[0].method)
	assert.Equal(t, "OnRunComplete", cb.calls[1].method)
}

func TestEngine_ExecuteStepFailureRecordsMetrics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		invokeErr  error
		wantStatus int
	}{
		{
			name:       "business failure marks step failed",
			invokeErr:  errors.New("invoke failed"),
			wantStatus: int(schema.PipelineFailed),
		},
		{
			name:       "context canceled marks step cancelled",
			invokeErr:  context.Canceled,
			wantStatus: int(schema.PipelineCancel),
		},
		{
			name:       "deadline exceeded marks step cancelled",
			invokeErr:  context.DeadlineExceeded,
			wantStatus: int(schema.PipelineCancel),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			op := "fail-op-" + strings.ReplaceAll(tt.name, " ", "-")
			registerExampleInvoker(t, op, func(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
				return nil, tt.invokeErr
			})
			store := newMockPipelineStore()
			pc := metrics.NewPipelineCollector(nil)
			def := Definition{
				Name: "fail-pl", Enabled: true, Trigger: Trigger{Event: "e1"},
				Steps: []Step{{Name: "bad", Capability: hub.CapExample, Operation: op}},
			}
			e := NewEngine([]Definition{def}, store, nil, pc, noopEC)
			defer e.Stop()
			err := e.executePipeline(context.Background(), def, types.DataEvent{EventID: "e1", EventType: "e1"}, "event")
			require.Error(t, err)
			require.NotEmpty(t, store.stepRuns)
			for _, sr := range store.stepRuns {
				assert.Equal(t, tt.wantStatus, sr.Status)
			}
			require.Len(t, store.runs, 1)
			for _, run := range store.runs {
				assert.Equal(t, tt.wantStatus, run.Status)
			}
		})
	}
}

func TestCheckpointDataJSONRoundTrip(t *testing.T) {
	t.Parallel()
	cp := CheckpointData{
		StepIndex: 1,
		StepResults: map[string]*StepResult{
			"s1": {Name: "s1", Output: map[string]any{"k": "v"}},
		},
		Event: types.DataEvent{EventID: "e1", EventType: "t"},
	}
	raw, err := sonic.Marshal(cp)
	require.NoError(t, err)
	var decoded CheckpointData
	require.NoError(t, sonic.Unmarshal(raw, &decoded))
	assert.Equal(t, cp.StepIndex, decoded.StepIndex)
}
