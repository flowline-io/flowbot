package partials

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestFormatDurationMs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{name: "sub-second", ms: 847, want: "847ms"},
		{name: "exact second", ms: 1000, want: "1.0s"},
		{name: "multi-second", ms: 1250, want: "1.2s"},
		{name: "zero", ms: 0, want: "0ms"},
		{name: "three-point-two seconds", ms: 3200, want: "3.2s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, FormatDurationMs(tt.ms))
		})
	}
}

func TestRunLatencyDisplayHelpers(t *testing.T) {
	t.Parallel()
	stats := &types.RunLatencyStats{
		SuccessRate: 0.6,
		P50Ms:       847,
		P95Ms:       3200,
		Total:       10,
	}
	tests := []struct {
		name string
		fn   func(*types.RunLatencyStats) string
		in   *types.RunLatencyStats
		want string
	}{
		{name: "success nil", fn: RunLatencySuccessOrDash, in: nil, want: "—"},
		{name: "success zero total", fn: RunLatencySuccessOrDash, in: &types.RunLatencyStats{}, want: "—"},
		{name: "success rate percent", fn: RunLatencySuccessOrDash, in: stats, want: "60%"},
		{name: "p50 nil", fn: RunLatencyP50OrDash, in: nil, want: "—"},
		{name: "p50 value", fn: RunLatencyP50OrDash, in: stats, want: "847ms"},
		{name: "p95 value", fn: RunLatencyP95OrDash, in: stats, want: "3.2s"},
		{name: "compact nil", fn: RunLatencyCompactOrDash, in: nil, want: "—"},
		{name: "compact value", fn: RunLatencyCompactOrDash, in: stats, want: "60% · 847ms / 3.2s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.fn(tt.in))
		})
	}
}

func TestAttachRunLatencyStats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		entries []PipelineListEntry
		stats   map[string]types.RunLatencyStats
		check   func(t *testing.T, got []PipelineListEntry)
	}{
		{
			name: "attaches matching stats",
			entries: []PipelineListEntry{
				{Definition: &gen.PipelineDefinition{Name: "a"}},
				{Definition: &gen.PipelineDefinition{Name: "b"}},
			},
			stats: map[string]types.RunLatencyStats{
				"a": {SuccessRate: 1, P50Ms: 100, P95Ms: 200, Total: 3},
			},
			check: func(t *testing.T, got []PipelineListEntry) {
				assert.NotNil(t, got[0].Stats)
				assert.Equal(t, int64(3), got[0].Stats.Total)
				assert.Nil(t, got[1].Stats)
			},
		},
		{
			name:    "nil stats map leaves entries unchanged",
			entries: []PipelineListEntry{{Definition: &gen.PipelineDefinition{Name: "x"}}},
			stats:   nil,
			check: func(t *testing.T, got []PipelineListEntry) {
				assert.Nil(t, got[0].Stats)
			},
		},
		{
			name:    "empty entries",
			entries: nil,
			stats:   map[string]types.RunLatencyStats{"a": {Total: 1}},
			check: func(t *testing.T, got []PipelineListEntry) {
				assert.Empty(t, got)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := AttachRunLatencyStats(tt.entries, tt.stats)
			tt.check(t, got)
		})
	}
}

func TestAttachWorkflowRunLatencyStats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		entries []WorkflowListEntry
		stats   map[string]types.RunLatencyStats
		wantA   bool
	}{
		{
			name:    "attaches matching workflow stats",
			entries: []WorkflowListEntry{{Name: "wf-a"}, {Name: "wf-b"}},
			stats:   map[string]types.RunLatencyStats{"wf-a": {Total: 2, SuccessRate: 1, P50Ms: 10, P95Ms: 20}},
			wantA:   true,
		},
		{
			name:    "nil stats",
			entries: []WorkflowListEntry{{Name: "wf-a"}},
			stats:   nil,
			wantA:   false,
		},
		{
			name:    "empty entries",
			entries: nil,
			stats:   map[string]types.RunLatencyStats{"wf-a": {Total: 1}},
			wantA:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := AttachWorkflowRunLatencyStats(tt.entries, tt.stats)
			if len(tt.entries) == 0 {
				assert.Empty(t, got)
				return
			}
			if tt.wantA {
				assert.NotNil(t, got[0].Stats)
			} else {
				assert.Nil(t, got[0].Stats)
			}
		})
	}
}
