package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestBuildDAG(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		tasks       []types.WorkflowTask
		wantErr     bool
		errContains string
		check       func(t *testing.T, nodes map[string]*dagNode, ready []string)
	}{
		{
			name: "linear-chain",
			tasks: []types.WorkflowTask{
				{ID: "a"},
				{ID: "b", Conn: []string{"a"}},
				{ID: "c", Conn: []string{"b"}},
			},
			check: func(t *testing.T, nodes map[string]*dagNode, ready []string) {
				assert.Equal(t, []string{"a"}, ready)
				assert.Equal(t, 0, nodes["a"].inDegree)
				assert.Equal(t, 1, nodes["b"].inDegree)
				assert.Equal(t, 1, nodes["c"].inDegree)
				assert.Equal(t, []string{"b"}, nodes["a"].deps)
				assert.Equal(t, []string{"c"}, nodes["b"].deps)
				assert.Empty(t, nodes["c"].deps)
			},
		},
		{
			name: "diamond-dag",
			tasks: []types.WorkflowTask{
				{ID: "a", Conn: []string{"b", "c"}},
				{ID: "b", Conn: []string{"d"}},
				{ID: "c", Conn: []string{"d"}},
				{ID: "d"},
			},
			check: func(t *testing.T, nodes map[string]*dagNode, ready []string) {
				assert.Equal(t, []string{"d"}, ready)
				assert.Equal(t, 0, nodes["d"].inDegree)
				assert.Equal(t, 1, nodes["b"].inDegree)
				assert.Equal(t, 1, nodes["c"].inDegree)
				assert.Equal(t, 2, nodes["a"].inDegree)
				assert.ElementsMatch(t, []string{"a"}, nodes["b"].deps)
				assert.ElementsMatch(t, []string{"a"}, nodes["c"].deps)
				assert.ElementsMatch(t, []string{"b", "c"}, nodes["d"].deps)
			},
		},
		{
			name: "independent-tasks",
			tasks: []types.WorkflowTask{
				{ID: "a"},
				{ID: "b"},
				{ID: "c"},
			},
			check: func(t *testing.T, nodes map[string]*dagNode, ready []string) {
				assert.ElementsMatch(t, []string{"a", "b", "c"}, ready)
				for _, n := range nodes {
					assert.Equal(t, 0, n.inDegree)
				}
			},
		},
		{
			name: "single-node",
			tasks: []types.WorkflowTask{
				{ID: "solo"},
			},
			check: func(t *testing.T, nodes map[string]*dagNode, ready []string) {
				assert.Equal(t, []string{"solo"}, ready)
				assert.Equal(t, 0, nodes["solo"].inDegree)
				assert.Empty(t, nodes["solo"].deps)
			},
		},
		{
			name: "fan-out-fan-in",
			tasks: []types.WorkflowTask{
				{ID: "root"},
				{ID: "left", Conn: []string{"root"}},
				{ID: "right", Conn: []string{"root"}},
				{ID: "merge", Conn: []string{"left", "right"}},
			},
			check: func(t *testing.T, nodes map[string]*dagNode, ready []string) {
				assert.Equal(t, []string{"root"}, ready)
				assert.Equal(t, 2, nodes["merge"].inDegree)
				assert.ElementsMatch(t, []string{"left", "right"}, nodes["root"].deps)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			nodes, ready, err := buildDAG(tt.tasks)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, nodes, ready)
			}
		})
	}
}

func TestRunParallelBasic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wf      types.WorkflowMetadata
		wantErr bool
	}{
		{
			name: "two-independent-mapper-tasks",
			wf: types.WorkflowMetadata{
				Name:           "parallel-mapper",
				MaxConcurrency: 2,
				Pipeline:       []string{"a", "b"},
				Tasks: []types.WorkflowTask{
					{ID: "a", Action: "mapper:", Params: types.KV{"out": "value-a"}},
					{ID: "b", Action: "mapper:", Params: types.KV{"out": "value-b"}},
				},
			},
		},
		{
			name: "three-all-independent",
			wf: types.WorkflowMetadata{
				Name:           "three-parallel",
				MaxConcurrency: 3,
				Pipeline:       []string{"a", "b", "c"},
				Tasks: []types.WorkflowTask{
					{ID: "a", Action: "mapper:", Params: types.KV{"out": "a"}},
					{ID: "b", Action: "mapper:", Params: types.KV{"out": "b"}},
					{ID: "c", Action: "mapper:", Params: types.KV{"out": "c"}},
				},
			},
		},
		{
			name: "diamond-dag-mapper",
			wf: types.WorkflowMetadata{
				Name:           "diamond-mapper",
				MaxConcurrency: 2,
				Pipeline:       []string{"d", "b", "c", "a"},
				Tasks: []types.WorkflowTask{
					{ID: "a", Action: "mapper:", Params: types.KV{"merged": `{{step "b" "result"}}|{{step "c" "result"}}`}, Conn: []string{"b", "c"}},
					{ID: "b", Action: "mapper:", Params: types.KV{"from": `{{step "d" "result"}}`}, Conn: []string{"d"}},
					{ID: "c", Action: "mapper:", Params: types.KV{"from": `{{step "d" "result"}}`}, Conn: []string{"d"}},
					{ID: "d", Action: "mapper:", Params: types.KV{"start": "root"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := NewRunner()
			err := runner.Execute(context.Background(), tt.wf, nil, "")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunParallelFailFast(t *testing.T) {
	t.Parallel()
	wf := types.WorkflowMetadata{
		Name:           "fail-fast-test",
		MaxConcurrency: 2,
		Pipeline:       []string{"failer", "mapper"},
		Tasks: []types.WorkflowTask{
			{ID: "failer", Action: "mapper:", Params: types.KV{"bad": "{{if}"}},
			{ID: "mapper", Action: "mapper:", Params: types.KV{"out": "should-complete-or-be-cancelled"}},
		},
	}
	runner := NewRunner()
	err := runner.Execute(context.Background(), wf, nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve params step failer")
}

func TestRunParallelEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wf      types.WorkflowMetadata
		wantErr bool
	}{
		{
			name: "max-concurrency-zero-should-be-sequential",
			wf: types.WorkflowMetadata{
				Name:           "sequential-fallback",
				MaxConcurrency: 0,
				Pipeline:       []string{"a", "b", "c"},
				Tasks: []types.WorkflowTask{
					{ID: "a", Action: "mapper:", Params: types.KV{"out": "a"}},
					{ID: "b", Action: "mapper:", Params: types.KV{"out": "b"}, Conn: []string{"a"}},
					{ID: "c", Action: "mapper:", Params: types.KV{"out": "c"}, Conn: []string{"b"}},
				},
			},
		},
		{
			name: "single-node-dag",
			wf: types.WorkflowMetadata{
				Name:           "single-node",
				MaxConcurrency: 5,
				Pipeline:       []string{"solo"},
				Tasks: []types.WorkflowTask{
					{ID: "solo", Action: "mapper:", Params: types.KV{"out": "done"}},
				},
			},
		},
		{
			name: "all-independent-max-conc-1-runs-sequential",
			wf: types.WorkflowMetadata{
				Name:           "forced-sequential",
				MaxConcurrency: 1,
				Pipeline:       []string{"a", "b"},
				Tasks: []types.WorkflowTask{
					{ID: "a", Action: "mapper:", Params: types.KV{"out": "a"}},
					{ID: "b", Action: "mapper:", Params: types.KV{"out": "b"}},
				},
			},
		},
		{
			name: "diamond-with-max-conc-2",
			wf: types.WorkflowMetadata{
				Name:           "diamond-conc-2",
				MaxConcurrency: 2,
				Pipeline:       []string{"d", "b", "c", "a"},
				Tasks: []types.WorkflowTask{
					{ID: "a", Action: "mapper:", Params: types.KV{"merged": `{{step "b" "result"}}|{{step "c" "result"}}`}, Conn: []string{"b", "c"}},
					{ID: "b", Action: "mapper:", Params: types.KV{"from": `{{step "d" "result"}}`}, Conn: []string{"d"}},
					{ID: "c", Action: "mapper:", Params: types.KV{"from": `{{step "d" "result"}}`}, Conn: []string{"d"}},
					{ID: "d", Action: "mapper:", Params: types.KV{"start": "root"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := NewRunner()
			err := runner.Execute(context.Background(), tt.wf, nil, "")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
