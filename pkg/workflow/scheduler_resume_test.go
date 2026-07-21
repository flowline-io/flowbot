package workflow

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestPremarkCompletedTasksForResume(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cp        CheckpointData
		tasks     []types.WorkflowTask
		wantReady []string
	}{
		{
			name: "single completed task unlocks dependent",
			cp: CheckpointData{
				CompletedTasks: map[string]bool{"root": true},
				StepResults:    map[string]string{"root": `{"start":"ok"}`},
			},
			tasks: []types.WorkflowTask{
				{ID: "leaf", Conn: []string{"root"}},
				{ID: "root"},
			},
			wantReady: []string{"leaf"},
		},
		{
			name: "unknown completed task is ignored",
			cp: CheckpointData{
				CompletedTasks: map[string]bool{"missing": true},
			},
			tasks: []types.WorkflowTask{
				{ID: "a"},
				{ID: "b", Conn: []string{"a"}},
			},
			wantReady: []string{"a"},
		},
		{
			name: "multiple completed tasks reduce in-degrees",
			cp: CheckpointData{
				CompletedTasks: map[string]bool{"a": true, "b": true},
				StepResults:    map[string]string{"a": "1", "b": "2"},
			},
			tasks: []types.WorkflowTask{
				{ID: "merge", Conn: []string{"a", "b"}},
				{ID: "a"},
				{ID: "b"},
			},
			wantReady: []string{"merge"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			nodes, _, err := buildDAG(tt.tasks)
			require.NoError(t, err)
			runner := NewRunner()
			runner.premarkCompletedTasksForResume(tt.cp, nodes)
			ready := runner.recomputeReadyList(types.WorkflowMetadata{Tasks: tt.tasks}, tt.cp, nodes, nil)
			assert.ElementsMatch(t, tt.wantReady, ready)
		})
	}
}

func TestCountRemainingTasksOnResume(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cp   CheckpointData
		wf   types.WorkflowMetadata
		want int
	}{
		{
			name: "none completed",
			cp:   CheckpointData{CompletedTasks: map[string]bool{}},
			wf: types.WorkflowMetadata{
				Tasks: []types.WorkflowTask{{ID: "a"}, {ID: "b"}},
			},
			want: 2,
		},
		{
			name: "one completed",
			cp:   CheckpointData{CompletedTasks: map[string]bool{"a": true}},
			wf: types.WorkflowMetadata{
				Tasks: []types.WorkflowTask{{ID: "a"}, {ID: "b"}},
			},
			want: 1,
		},
		{
			name: "all completed",
			cp:   CheckpointData{CompletedTasks: map[string]bool{"a": true, "b": true}},
			wf: types.WorkflowMetadata{
				Tasks: []types.WorkflowTask{{ID: "a"}, {ID: "b"}},
			},
			want: 0,
		},
	}

	runner := NewRunner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, runner.countRemainingTasksOnResume(tt.wf, tt.cp))
		})
	}
}

func TestFinalizeParallelStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		runID   int64
		run     *gen.WorkflowRun
		err     error
		store   bool
		wantLog int
	}{
		{
			name:    "nil store returns error unchanged",
			runID:   1,
			err:     assert.AnError,
			store:   false,
			wantLog: 0,
		},
		{
			name:  "success updates done status",
			runID: 1,
			store: true,
		},
		{
			name:  "failure updates failed status",
			runID: 1,
			err:   assert.AnError,
			store: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var store *mockWorkflowStore
			runner := NewRunner()
			if tt.store {
				store = newMockWorkflowStore()
				runner = NewRunnerWithStore(store, nil, nil, "", "")
				if tt.run == nil {
					run, err := store.CreateRun(context.Background(), 0, "wf", "f.yaml", "manual", nil, nil)
					require.NoError(t, err)
					tt.run = run
				}
			}
			ret := runner.finalizeParallelStatus(context.Background(), tt.run, tt.runID, tt.err)
			if tt.err != nil {
				require.Error(t, ret)
			} else if tt.store {
				require.NoError(t, ret)
				assert.Contains(t, store.statusLog, int(schema.WorkflowRunDone))
			}
		})
	}
}

func TestRunner_ParallelResume(t *testing.T) {
	t.Parallel()
	wfContent := `name: parallel-resume
max_concurrency: 2
resumable: true
pipeline:
  - a
  - b
  - c
tasks:
  - id: a
    action: "mapper:"
    params:
      out: a
  - id: b
    action: "mapper:"
    params:
      out: b
  - id: c
    action: "mapper:"
    params:
      merged: '{{step "a" "result"}}|{{step "b" "result"}}'
    conn: [a, b]
`
	wf, err := ParseYAML([]byte(wfContent))
	require.NoError(t, err)
	store := newMockWorkflowStore()
	run, err := store.CreateRun(context.Background(), 0, "parallel-resume", "db", "manual", nil, nil)
	require.NoError(t, err)
	run.Status = int(schema.WorkflowRunFailed)
	require.NoError(t, store.SaveCheckpoint(context.Background(), run.ID, &CheckpointData{
		CompletedTasks: map[string]bool{"a": true},
		StepResults:    map[string]string{"a": `{"out":"a"}`},
		Input:          types.KV{},
	}))

	runner := NewRunnerWithStore(store, nil, nil, "", "").WithDefinitionStore(newMockDefinitionStore(wf))
	err = runner.ResumeWorkflow(context.Background(), run.ID)
	require.NoError(t, err)
	assert.Contains(t, store.statusLog, int(schema.WorkflowRunDone))
}

func TestRunParallelResumeAllComplete(t *testing.T) {
	t.Parallel()
	store := newMockWorkflowStore()
	run, err := store.CreateRun(context.Background(), 0, "done-wf", "f.yaml", "manual", nil, nil)
	require.NoError(t, err)

	wf := types.WorkflowMetadata{
		Name:           "done-wf",
		MaxConcurrency: 2,
		Tasks: []types.WorkflowTask{
			{ID: "a", Action: "mapper:", Params: types.KV{"k": "v"}},
		},
	}
	cp := CheckpointData{
		CompletedTasks: map[string]bool{"a": true},
		StepResults:    map[string]string{"a": `{"k":"v"}`},
	}

	runner := NewRunnerWithStore(store, nil, nil, "", "")
	err = runner.runParallelResume(context.Background(), run.ID, wf, cp)
	require.NoError(t, err)
	assert.Contains(t, store.statusLog, int(schema.WorkflowRunDone))
}

func TestRunParallelResumeTaskHandler(t *testing.T) {
	registerWorkflowCapabilityInvoker(t)

	wf := types.WorkflowMetadata{
		Name:           "resume-handler",
		MaxConcurrency: 2,
		Resumable:      true,
		Tasks: []types.WorkflowTask{
			{ID: "cap1", Action: "capability:example.echo", Params: types.KV{"value": "resume"}},
		},
	}
	nodes, ready, err := buildDAG(wf.Tasks)
	require.NoError(t, err)

	store := newMockWorkflowStore()
	run, err := store.CreateRun(context.Background(), 0, wf.Name, "f.yaml", "manual", nil, nil)
	require.NoError(t, err)

	runner := NewRunnerWithStore(store, nil, nil, "", "")
	results := make(map[string]string)
	var mu sync.RWMutex
	var firstErr error
	var errOnce sync.Once
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner.runParallelResumeTaskHandler(
		ctx, "cap1", wf.Tasks[0], nodes, nil, &results, &mu, run, &ready, &wf,
		&firstErr, &errOnce, cancel,
	)
	require.NoError(t, firstErr)
	assert.NotEmpty(t, results)
}

func TestExecuteMapperStepMarshalErrorParallel(t *testing.T) {
	t.Parallel()
	store := newMockWorkflowStore()
	runner := NewRunnerWithStore(store, nil, nil, "", "")
	run, err := store.CreateRun(context.Background(), 0, "wf", "f.yaml", "manual", nil, nil)
	require.NoError(t, err)
	stepRun, err := store.CreateStepRun(context.Background(), run.ID, "m1", "", "mapper:", "mapper", nil, 1)
	require.NoError(t, err)

	results := make(map[string]string)
	var mu sync.RWMutex
	err = runner.executeMapperStep(context.Background(), "m1", types.KV{"bad": make(chan int)}, &results, &mu, stepRun)
	require.Error(t, err)
	assert.Equal(t, int(schema.WorkflowRunFailed), store.stepRuns[stepRun.ID].Status)
}
