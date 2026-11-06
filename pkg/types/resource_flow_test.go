package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeFlowGraphWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)

	t.Run("default 7d", func(t *testing.T) {
		since, until := NormalizeFlowGraphWindow(time.Time{}, time.Time{}, 0, now)
		assert.Equal(t, now, until)
		assert.Equal(t, now.Add(-FlowGraphDefaultWindow), since)
	})

	t.Run("retention 30 caps below 90", func(t *testing.T) {
		since, until := NormalizeFlowGraphWindow(now.Add(-120*24*time.Hour), now, 30, now)
		assert.Equal(t, now, until)
		assert.Equal(t, now.Add(-30*24*time.Hour), since)
	})

	t.Run("unlimited retention still caps at 90", func(t *testing.T) {
		since, until := NormalizeFlowGraphWindow(now.Add(-200*24*time.Hour), now, 0, now)
		assert.Equal(t, now, until)
		assert.Equal(t, now.Add(-90*24*time.Hour), since)
	})
}

func TestNormalizeLineageTraceDepthAndDirection(t *testing.T) {
	t.Parallel()
	assert.Equal(t, LineageTraceDirectionUp, NormalizeLineageTraceDirection(""))
	assert.Equal(t, LineageTraceDirectionBoth, NormalizeLineageTraceDirection(LineageTraceDirectionBoth))
	assert.Equal(t, LineageTraceDefaultDepth, NormalizeLineageTraceDepth(0))
	assert.Equal(t, LineageTraceMaxDepth, NormalizeLineageTraceDepth(99))
	assert.Equal(t, 3, NormalizeLineageTraceDepth(3))
}

func TestCapFlowGraph(t *testing.T) {
	t.Parallel()
	g := &FlowGraph{
		Nodes: []FlowGraphNode{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
		Edges: []FlowGraphEdge{
			{Source: "a", Target: "b", CountOK: 1},
			{Source: "b", Target: "c", CountOK: 10},
			{Source: "c", Target: "d", CountFailed: 2},
			{Source: "a", Target: "d", CountOK: 5},
		},
	}
	out := CapFlowGraph(g, 3, 2)
	requireLen := func(n int) {
		t.Helper()
		if len(out.Edges) != n {
			t.Fatalf("edges=%d want %d", len(out.Edges), n)
		}
	}
	requireLen(2)
	assert.Equal(t, int64(2), out.Edges[0].CountFailed)
	assert.LessOrEqual(t, len(out.Nodes), 3)
	assert.NotNil(t, CapFlowGraph(nil, 20, 50))
}
