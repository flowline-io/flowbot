package partials

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

func TestBuildRunWaterfall(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	end1 := base.Add(2 * time.Second)
	start2 := base.Add(2 * time.Second)
	end2 := base.Add(5 * time.Second)

	tests := []struct {
		name  string
		steps []runWaterfallInput
		want  []RunWaterfallBar
	}{
		{
			name:  "empty steps",
			steps: nil,
			want:  nil,
		},
		{
			name: "single completed step fills full width",
			steps: []runWaterfallInput{{
				Name: "a", Status: int(schema.PipelineDone),
				StartedAt: base, CompletedAt: &end1,
			}},
			want: []RunWaterfallBar{{
				Name: "a", Status: int(schema.PipelineDone),
				OffsetPct: 0, WidthPct: 100,
			}},
		},
		{
			name: "sequential steps share timeline",
			steps: []runWaterfallInput{
				{Name: "a", Status: int(schema.PipelineDone), StartedAt: base, CompletedAt: &end1},
				{Name: "b", Status: int(schema.PipelineFailed), StartedAt: start2, CompletedAt: &end2},
			},
			want: []RunWaterfallBar{
				{Name: "a", Status: int(schema.PipelineDone), OffsetPct: 0, WidthPct: 40},
				{Name: "b", Status: int(schema.PipelineFailed), OffsetPct: 40, WidthPct: 60},
			},
		},
		{
			name: "incomplete step uses now as end",
			steps: []runWaterfallInput{
				{Name: "running", Status: int(schema.PipelineStart), StartedAt: base, CompletedAt: nil},
			},
			want: []RunWaterfallBar{{
				Name: "running", Status: int(schema.PipelineStart),
				OffsetPct: 0, WidthPct: 100,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildRunWaterfall(tt.steps, base.Add(10*time.Second))
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.Equal(t, tt.want[i].Name, got[i].Name)
				assert.Equal(t, tt.want[i].Status, got[i].Status)
				assert.InDelta(t, tt.want[i].OffsetPct, got[i].OffsetPct, 0.5)
				assert.InDelta(t, tt.want[i].WidthPct, got[i].WidthPct, 0.5)
			}
		})
	}
}

func TestPipelineStepErrorSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		steps []*gen.PipelineStepRun
		want  []RunErrorSummaryItem
	}{
		{name: "no failures", steps: []*gen.PipelineStepRun{{StepName: "ok", Status: 2}}, want: nil},
		{
			name: "failed with error text",
			steps: []*gen.PipelineStepRun{
				{StepName: "ok", Status: 2},
				{StepName: "boom", Status: 4, Error: "timeout"},
			},
			want: []RunErrorSummaryItem{{Name: "boom", Error: "timeout"}},
		},
		{
			name: "error field without failed status still summarized",
			steps: []*gen.PipelineStepRun{
				{StepName: "warn", Status: 2, Error: "partial"},
			},
			want: []RunErrorSummaryItem{{Name: "warn", Error: "partial"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, PipelineStepErrorSummary(tt.steps))
		})
	}
}

func TestPipelineStepHasDetailAndOpen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		step      *gen.PipelineStepRun
		hasDetail bool
		open      bool
	}{
		{name: "nil", step: nil, hasDetail: false, open: false},
		{name: "empty success", step: &gen.PipelineStepRun{Status: 2}, hasDetail: false, open: false},
		{name: "params only", step: &gen.PipelineStepRun{Params: map[string]any{"a": 1}, Status: 2}, hasDetail: true, open: false},
		{name: "failed with error", step: &gen.PipelineStepRun{Status: 4, Error: "x"}, hasDetail: true, open: true},
		{name: "failed status alone", step: &gen.PipelineStepRun{Status: 4}, hasDetail: true, open: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.hasDetail, pipelineStepHasDetail(tt.step))
			assert.Equal(t, tt.open, pipelineStepDetailOpen(tt.step))
		})
	}
}

func TestTruncateErrorSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "short unchanged", input: "boom", want: "boom"},
		{name: "empty", input: "", want: ""},
		{name: "long truncated", input: string(make([]byte, 200)), want: string(make([]byte, 160)) + "…"},
	}
	// Fill long with readable chars
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	tests[2].input = string(long)
	tests[2].want = string(long[:160]) + "…"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, TruncateErrorSummary(tt.input))
		})
	}
}

func TestWaterfallBarClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		fn     func(int) string
		status int
		want   string
	}{
		{name: "pipeline success", fn: PipelineWaterfallBarClass, status: 2, want: "run-waterfall-bar run-waterfall-bar-success"},
		{name: "pipeline failed", fn: PipelineWaterfallBarClass, status: 4, want: "run-waterfall-bar run-waterfall-bar-error"},
		{name: "pipeline running", fn: PipelineWaterfallBarClass, status: 1, want: "run-waterfall-bar run-waterfall-bar-running"},
		{name: "workflow failed status 3", fn: WorkflowWaterfallBarClass, status: 3, want: "run-waterfall-bar run-waterfall-bar-error"},
		{name: "pipeline cancel muted", fn: PipelineWaterfallBarClass, status: 3, want: "run-waterfall-bar run-waterfall-bar-muted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.fn(tt.status))
		})
	}
}
