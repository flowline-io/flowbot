package workflow

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

type mockWorkflowStore struct {
	mu          sync.Mutex
	runs        map[int64]*gen.WorkflowRun
	stepRuns    map[int64]*gen.WorkflowStepRun
	checkpoints map[int64]*CheckpointData
	nextRunID   int64
	nextStepID  int64
	heartbeats  int
	statusLog   []int
	stepStatus  []int
}

func newMockWorkflowStore() *mockWorkflowStore {
	return &mockWorkflowStore{
		runs:        make(map[int64]*gen.WorkflowRun),
		stepRuns:    make(map[int64]*gen.WorkflowStepRun),
		checkpoints: make(map[int64]*CheckpointData),
	}
}

func (m *mockWorkflowStore) CreateRun(_ context.Context, workflowID int64, workflowName, workflowFile, triggerType string, triggerInfo, inputParams map[string]any) (*gen.WorkflowRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextRunID++
	run := &gen.WorkflowRun{
		ID:           m.nextRunID,
		WorkflowName: workflowName,
		WorkflowFile: workflowFile,
		TriggerType:  triggerType,
		TriggerInfo:  triggerInfo,
		InputParams:  inputParams,
		Status:       int(schema.WorkflowRunRunning),
	}
	if workflowID != 0 {
		run.WorkflowID = &workflowID
	}
	m.runs[run.ID] = run
	return run, nil
}

type mockDefinitionStore struct {
	byName map[string]*types.WorkflowMetadata
	err    error
}

func newMockDefinitionStore(defs ...*types.WorkflowMetadata) *mockDefinitionStore {
	m := &mockDefinitionStore{byName: make(map[string]*types.WorkflowMetadata)}
	for _, d := range defs {
		if d != nil {
			m.byName[d.Name] = d
		}
	}
	return m
}

func (m *mockDefinitionStore) GetMetadata(_ context.Context, name string) (*types.WorkflowMetadata, error) {
	if m != nil && m.err != nil {
		return nil, m.err
	}
	if m == nil || m.byName == nil {
		return nil, assert.AnError
	}
	wf, ok := m.byName[name]
	if !ok {
		return nil, assert.AnError
	}
	return wf, nil
}

func (m *mockWorkflowStore) UpdateRunStatus(ctx context.Context, runID int64, status int, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusLog = append(m.statusLog, status)
	if run, ok := m.runs[runID]; ok {
		run.Status = status
	}
	return nil
}

func (m *mockWorkflowStore) CreateStepRun(_ context.Context, runID int64, stepID, stepName, action, actionType string, params map[string]any, attempt int) (*gen.WorkflowStepRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextStepID++
	sr := &gen.WorkflowStepRun{
		ID:            m.nextStepID,
		WorkflowRunID: runID,
		StepID:        stepID,
		StepName:      stepName,
		Action:        action,
		ActionType:    actionType,
		Params:        params,
		Attempt:       attempt,
		Status:        int(schema.WorkflowRunRunning),
	}
	m.stepRuns[sr.ID] = sr
	return sr, nil
}

func (m *mockWorkflowStore) UpdateStepRun(ctx context.Context, stepRunID int64, status int, result map[string]any, _ string, _ int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stepStatus = append(m.stepStatus, status)
	if sr, ok := m.stepRuns[stepRunID]; ok {
		sr.Status = status
		sr.Result = result
	}
	return nil
}

func (m *mockWorkflowStore) SaveCheckpoint(_ context.Context, runID int64, data any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp, ok := data.(*CheckpointData)
	if !ok {
		return nil
	}
	copyCP := *cp
	m.checkpoints[runID] = &copyCP
	return nil
}

func (m *mockWorkflowStore) GetIncompleteRuns(context.Context) ([]*gen.WorkflowRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil, nil
}

func (m *mockWorkflowStore) GetCheckpoint(_ context.Context, runID int64, target any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp, ok := m.checkpoints[runID]
	if !ok {
		return assert.AnError
	}
	dest, ok := target.(*CheckpointData)
	if !ok {
		return assert.AnError
	}
	*dest = *cp
	return nil
}

func (m *mockWorkflowStore) GetRun(_ context.Context, runID int64) (*gen.WorkflowRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.runs[runID]
	if !ok {
		return nil, assert.AnError
	}
	return run, nil
}

func (m *mockWorkflowStore) UpdateRunHeartbeat(context.Context, int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.heartbeats++
	return nil
}

func writeWorkflowFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestRunner_Close(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "close default runner"},
		{name: "close runner with store"},
		{name: "double close is safe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var r *Runner
			switch tt.name {
			case "close runner with store":
				r = NewRunnerWithStore(newMockWorkflowStore(), nil, nil, "", "")
			default:
				r = NewRunner()
			}
			require.NoError(t, r.Close())
			if tt.name == "double close is safe" {
				require.NoError(t, r.Close())
			}
		})
	}
}

func TestRunner_Run(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(t *testing.T) (*Runner, *types.Task)
		wantErr     bool
		errContains string
	}{
		{
			name: "missing task id returns validation error",
			setup: func(_ *testing.T) (*Runner, *types.Task) {
				return NewRunner(), &types.Task{Run: "shell:echo hello"}
			},
			wantErr:     true,
			errContains: "task id is required",
		},
		{
			name: "capability runtime missing engine",
			setup: func(t *testing.T) (*Runner, *types.Task) {
				r := NewRunner()
				delete(r.engines, runtime.Capability)
				task, err := WorkflowTaskToTask(types.WorkflowTask{ID: "s1", Action: "capability:example.list"})
				require.NoError(t, err)
				return r, task
			},
			wantErr:     true,
			errContains: "unknown runtime type",
		},
		{
			name: "unknown runtime when engine removed",
			setup: func(t *testing.T) (*Runner, *types.Task) {
				r := NewRunner()
				delete(r.engines, runtime.Shell)
				task, err := WorkflowTaskToTask(types.WorkflowTask{ID: "s1", Action: "shell:echo hi"})
				require.NoError(t, err)
				return r, task
			},
			wantErr:     true,
			errContains: "unknown runtime type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner, task := tt.setup(t)
			defer runner.Close()
			err := runner.Run(context.Background(), task)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRunner_AuditEventsOnExecute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wf         types.WorkflowMetadata
		wantEvents []string
	}{
		{
			name: "mapper workflow emits start and complete",
			wf: types.WorkflowMetadata{
				Name:     "audit-wf",
				Pipeline: []string{"m1"},
				Tasks:    []types.WorkflowTask{{ID: "m1", Action: "mapper:", Params: types.KV{"k": "v"}}},
			},
			wantEvents: []string{"workflow.start", "workflow.complete"},
		},
		{
			name: "invalid template emits start and fail",
			wf: types.WorkflowMetadata{
				Name:     "fail-wf",
				Pipeline: []string{"s1"},
				Tasks:    []types.WorkflowTask{{ID: "s1", Action: "mapper:", Params: types.KV{"bad": "{{if}"}}},
			},
			wantEvents: []string{"workflow.start", "workflow.fail"},
		},
		{
			name: "sequential mapper success emits start and complete",
			wf: types.WorkflowMetadata{
				Name:     "mapper-wf",
				Pipeline: []string{"m1"},
				Tasks:    []types.WorkflowTask{{ID: "m1", Action: "mapper:", Params: types.KV{"k": "v"}}},
			},
			wantEvents: []string{"workflow.start", "workflow.complete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			auditor := &mockAuditor{}
			runner := NewRunnerWithStore(nil, auditor, nil, "", "")
			err := runner.Execute(context.Background(), tt.wf, nil, "")
			if tt.wantEvents[len(tt.wantEvents)-1] == "workflow.fail" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, auditor.entries, len(tt.wantEvents))
			for i, action := range tt.wantEvents {
				assert.Equal(t, action, auditor.entries[i].Action)
			}
		})
	}
}

func TestRunner_ExecuteWithStore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wf      types.WorkflowMetadata
		input   types.KV
		wantErr bool
		check   func(t *testing.T, store *mockWorkflowStore)
	}{
		{
			name: "resumable sequential saves checkpoints and completes",
			wf: types.WorkflowMetadata{
				Name:      "stored-wf",
				Resumable: true,
				Pipeline:  []string{"m1", "m2"},
				Tasks: []types.WorkflowTask{
					{ID: "m1", Action: "mapper:", Params: types.KV{"a": "1"}},
					{ID: "m2", Action: "mapper:", Params: types.KV{"b": `{{step "m1" "result"}}`}},
				},
			},
			input: types.KV{"seed": "input"},
			check: func(t *testing.T, store *mockWorkflowStore) {
				assert.NotEmpty(t, store.checkpoints)
				assert.Contains(t, store.statusLog, int(schema.WorkflowRunDone))
				assert.NotEmpty(t, store.stepRuns)
			},
		},
		{
			name: "mapper chain records step runs in store",
			wf: types.WorkflowMetadata{
				Name:     "mapper-stored",
				Pipeline: []string{"m1", "m2"},
				Tasks: []types.WorkflowTask{
					{ID: "m1", Action: "mapper:", Params: types.KV{"a": "1"}},
					{ID: "m2", Action: "mapper:", Params: types.KV{"b": `{{step "m1" "result"}}`}},
				},
			},
			check: func(t *testing.T, store *mockWorkflowStore) {
				require.Len(t, store.stepRuns, 2)
				for _, sr := range store.stepRuns {
					assert.Equal(t, int(schema.WorkflowRunDone), sr.Status)
				}
			},
		},
		{
			name: "missing task in pipeline fails run",
			wf: types.WorkflowMetadata{
				Name:     "missing-task",
				Pipeline: []string{"ghost"},
				Tasks:    []types.WorkflowTask{{ID: "other", Action: "mapper:", Params: types.KV{}}},
			},
			wantErr: true,
			check: func(t *testing.T, store *mockWorkflowStore) {
				assert.Contains(t, store.statusLog, int(schema.WorkflowRunFailed))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMockWorkflowStore()
			runner := NewRunnerWithStore(store, nil, nil, "file.yaml", "manual")
			err := runner.Execute(context.Background(), tt.wf, tt.input, "override.yaml")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t, store)
			}
		})
	}
}

func TestRunner_RunWithRetry(t *testing.T) {
	t.Parallel()
	store := newMockWorkflowStore()
	runner := NewRunnerWithStore(store, nil, nil, "", "")
	delete(runner.engines, runtime.Shell)

	task, err := WorkflowTaskToTask(types.WorkflowTask{ID: "s1", Action: "shell:noop"})
	require.NoError(t, err)

	run, err := store.CreateRun(context.Background(), 0, "retry-wf", "f.yaml", "manual", nil, nil)
	require.NoError(t, err)
	stepRun, err := store.CreateStepRun(context.Background(), run.ID, "s1", "", "shell:noop", "shell", nil, 1)
	require.NoError(t, err)

	attempt, err := runner.runWithRetry(context.Background(), task, &types.RetryConfig{
		MaxAttempts: 2,
		Delay:       time.Millisecond,
	}, "s1", stepRun)
	require.Error(t, err)
	assert.Equal(t, 2, attempt)
	assert.Contains(t, err.Error(), "unknown runtime type")
}

func TestRunner_ResumeWorkflow(t *testing.T) {
	t.Parallel()
	wfContent := `name: resume-wf
resumable: true
pipeline:
  - m1
  - m2
tasks:
  - id: m1
    action: "mapper:"
    params:
      done: "yes"
  - id: m2
    action: "mapper:"
    params:
      from_m1: '{{step "m1" "result"}}'
`
	wf, err := ParseYAML([]byte(wfContent))
	require.NoError(t, err)
	store := newMockWorkflowStore()
	run, err := store.CreateRun(context.Background(), 0, "resume-wf", "db", "manual", nil, nil)
	require.NoError(t, err)
	require.NoError(t, store.SaveCheckpoint(context.Background(), run.ID, &CheckpointData{
		StepIndex:   0,
		StepResults: map[string]string{"m1": `{"done":"yes"}`},
		Input:       types.KV{"seed": "v"},
	}))

	runner := NewRunnerWithStore(store, nil, nil, "", "").WithDefinitionStore(newMockDefinitionStore(wf))
	err = runner.ResumeWorkflow(context.Background(), run.ID)
	require.NoError(t, err)
	assert.Contains(t, store.statusLog, int(schema.WorkflowRunDone))
}

func TestRunner_ResumeWorkflow_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setup       func(t *testing.T) (*Runner, int64)
		errContains string
	}{
		{
			name: "no store configured",
			setup: func(_ *testing.T) (*Runner, int64) {
				return NewRunner(), 1
			},
			errContains: "cannot resume workflow without a store",
		},
		{
			name: "no definition store configured",
			setup: func(t *testing.T) (*Runner, int64) {
				store := newMockWorkflowStore()
				run, err := store.CreateRun(context.Background(), 0, "wf", "db", "manual", nil, nil)
				require.NoError(t, err)
				return NewRunnerWithStore(store, nil, nil, "", ""), run.ID
			},
			errContains: "cannot resume workflow without a definition store",
		},
		{
			name: "non-resumable status",
			setup: func(t *testing.T) (*Runner, int64) {
				store := newMockWorkflowStore()
				run, err := store.CreateRun(context.Background(), 0, "wf", "db", "manual", nil, nil)
				require.NoError(t, err)
				run.Status = int(schema.WorkflowRunDone)
				wf := &types.WorkflowMetadata{Name: "wf", Pipeline: []string{"m1"}, Tasks: []types.WorkflowTask{{ID: "m1", Action: "mapper:"}}}
				return NewRunnerWithStore(store, nil, nil, "", "").WithDefinitionStore(newMockDefinitionStore(wf)), run.ID
			},
			errContains: "not resumable",
		},
		{
			name: "missing checkpoint",
			setup: func(t *testing.T) (*Runner, int64) {
				store := newMockWorkflowStore()
				wf, err := ParseYAML([]byte(`name: wf
pipeline: [m1]
tasks:
  - id: m1
    action: "mapper:"
    params: {k: v}
`))
				require.NoError(t, err)
				run, err := store.CreateRun(context.Background(), 0, "wf", "db", "manual", nil, nil)
				require.NoError(t, err)
				run.Status = int(schema.WorkflowRunFailed)
				return NewRunnerWithStore(store, nil, nil, "", "").WithDefinitionStore(newMockDefinitionStore(wf)), run.ID
			},
			errContains: "get checkpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner, runID := tt.setup(t)
			err := runner.ResumeWorkflow(context.Background(), runID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestRunner_Heartbeat(t *testing.T) {
	t.Parallel()
	store := newMockWorkflowStore()
	runner := NewRunnerWithStore(store, nil, nil, "", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner.heartbeat(ctx, 1)
	assert.Equal(t, 0, store.heartbeats)
}

func TestResultCopy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  map[string]string
	}{
		{name: "empty map", src: map[string]string{}},
		{name: "single entry", src: map[string]string{"a": "1"}},
		{name: "multiple entries", src: map[string]string{"a": "1", "b": "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			copied := resultCopy(tt.src)
			assert.Equal(t, tt.src, copied)
			if len(tt.src) > 0 {
				copied["mut"] = "x"
				assert.NotEqual(t, tt.src["mut"], copied["mut"])
			}
		})
	}
}

func TestRunner_AuditWorkflowEventNilAuditor(t *testing.T) {
	t.Parallel()
	runner := NewRunner()
	runner.auditWorkflowEvent(context.Background(), "wf", "workflow.start")
}

func TestRunner_AuditWithAuditor(t *testing.T) {
	t.Parallel()
	auditor := &mockAuditor{}
	runner := NewRunnerWithStore(nil, auditor, nil, "", "")
	runner.auditWorkflowEvent(context.Background(), "named-wf", "workflow.start")
	require.Len(t, auditor.entries, 1)
	assert.Equal(t, "workflow", auditor.entries[0].Subject.SubjectType)
	assert.Equal(t, "named-wf", auditor.entries[0].Target.ID)
}

func registerWorkflowCapabilityInvoker(t *testing.T) {
	t.Helper()
	require.NoError(t, capability.RegisterInvoker(hub.CapExample, "echo", func(_ context.Context, params map[string]any) (*capability.InvokeResult, error) {
		return &capability.InvokeResult{
			Capability: hub.CapExample,
			Operation:  "echo",
			Data:       map[string]any{"value": params["value"]},
		}, nil
	}))
	t.Cleanup(func() {
		capability.UnregisterInvoker(hub.CapExample, "echo")
	})
}

func TestRunner_ExecuteCapabilityStep(t *testing.T) {
	registerWorkflowCapabilityInvoker(t)

	tests := []struct {
		name    string
		wf      types.WorkflowMetadata
		wantErr bool
		check   func(t *testing.T, store *mockWorkflowStore)
	}{
		{
			name: "sequential capability step succeeds",
			wf: types.WorkflowMetadata{
				Name:     "cap-seq",
				Pipeline: []string{"cap1"},
				Tasks: []types.WorkflowTask{
					{
						ID:     "cap1",
						Action: "capability:example.echo",
						Params: types.KV{"value": "hello"},
					},
				},
			},
			check: func(t *testing.T, store *mockWorkflowStore) {
				require.Len(t, store.stepRuns, 1)
				assert.Equal(t, int(schema.WorkflowRunDone), store.stepRuns[1].Status)
			},
		},
		{
			name: "parallel capability step succeeds",
			wf: types.WorkflowMetadata{
				Name:           "cap-par",
				MaxConcurrency: 2,
				Pipeline:       []string{"cap1"},
				Tasks: []types.WorkflowTask{
					{
						ID:     "cap1",
						Action: "capability:example.echo",
						Params: types.KV{"value": "parallel"},
					},
				},
			},
			check: func(t *testing.T, store *mockWorkflowStore) {
				require.NotEmpty(t, store.stepRuns)
			},
		},
		{
			name: "capability step failure records failed step",
			wf: types.WorkflowMetadata{
				Name:     "cap-fail",
				Pipeline: []string{"cap1"},
				Tasks: []types.WorkflowTask{
					{ID: "cap1", Action: "capability:example.missing"},
				},
			},
			wantErr: true,
			check: func(t *testing.T, store *mockWorkflowStore) {
				require.NotEmpty(t, store.stepRuns)
				for _, sr := range store.stepRuns {
					assert.Equal(t, int(schema.WorkflowRunFailed), sr.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockWorkflowStore()
			runner := NewRunnerWithStore(store, nil, nil, "", "")
			err := runner.Execute(context.Background(), tt.wf, nil, "")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t, store)
			}
		})
	}
}

func TestRunner_ResumeCapabilityStep(t *testing.T) {
	registerWorkflowCapabilityInvoker(t)

	wfContent := `name: resume-cap
resumable: true
pipeline:
  - m1
  - cap1
tasks:
  - id: m1
    action: "mapper:"
    params:
      seed: "v"
  - id: cap1
    action: capability:example.echo
    params:
      value: '{{step "m1" "result"}}'
`
	wf, err := ParseYAML([]byte(wfContent))
	require.NoError(t, err)
	store := newMockWorkflowStore()
	run, err := store.CreateRun(context.Background(), 0, "resume-cap", "db", "manual", nil, nil)
	require.NoError(t, err)
	run.Status = int(schema.WorkflowRunFailed)
	require.NoError(t, store.SaveCheckpoint(context.Background(), run.ID, &CheckpointData{
		StepIndex:   0,
		StepResults: map[string]string{"m1": `{"seed":"v"}`},
	}))

	runner := NewRunnerWithStore(store, nil, nil, "", "").WithDefinitionStore(newMockDefinitionStore(wf))
	err = runner.ResumeWorkflow(context.Background(), run.ID)
	require.NoError(t, err)
	assert.Contains(t, store.statusLog, int(schema.WorkflowRunDone))
}

func TestFailStepAndFailRun(t *testing.T) {
	t.Parallel()
	store := newMockWorkflowStore()
	runner := NewRunnerWithStore(store, nil, nil, "", "")
	run, err := store.CreateRun(context.Background(), 0, "wf", "f.yaml", "manual", nil, nil)
	require.NoError(t, err)
	stepRun, err := store.CreateStepRun(context.Background(), run.ID, "s1", "", "mapper:", "mapper", nil, 1)
	require.NoError(t, err)

	runner.failStep(context.Background(), stepRun, assert.AnError, 2)
	assert.Equal(t, int(schema.WorkflowRunFailed), store.stepRuns[stepRun.ID].Status)

	ctx, cancel := context.WithCancel(context.Background())
	runner.failRun(ctx, run, cancel, assert.AnError)
	assert.Contains(t, store.statusLog, int(schema.WorkflowRunFailed))
}

func TestExecuteSequentialMapperStepMarshalError(t *testing.T) {
	t.Parallel()
	store := newMockWorkflowStore()
	runner := NewRunnerWithStore(store, nil, nil, "", "")
	run, err := store.CreateRun(context.Background(), 0, "wf", "f.yaml", "manual", nil, nil)
	require.NoError(t, err)
	stepRun, err := store.CreateStepRun(context.Background(), run.ID, "m1", "", "mapper:", "mapper", nil, 1)
	require.NoError(t, err)

	results := make(map[string]string)
	err = runner.executeSequentialMapperStep(
		context.Background(),
		"m1",
		types.KV{"bad": make(chan int)},
		ParseAction("mapper:"),
		"wf",
		results,
		stepRun,
	)
	require.Error(t, err)
	assert.Equal(t, int(schema.WorkflowRunFailed), store.stepRuns[stepRun.ID].Status)
}
