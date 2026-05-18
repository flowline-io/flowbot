package workflow

import (
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
