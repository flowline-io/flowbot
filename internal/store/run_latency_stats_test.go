package store

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestComputeRunLatencyStats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		outcomes []runLatencyOutcome
		want     types.RunLatencyStats
	}{
		{
			name:     "empty outcomes returns zero stats",
			outcomes: nil,
			want:     types.RunLatencyStats{},
		},
		{
			name: "single success run",
			outcomes: []runLatencyOutcome{
				{durationMs: 1000, success: true},
			},
			want: types.RunLatencyStats{
				SuccessRate: 1.0,
				P50Ms:       1000,
				P95Ms:       1000,
				Total:       1,
			},
		},
		{
			name: "mixed success and failure with nearest-rank percentiles",
			outcomes: []runLatencyOutcome{
				{durationMs: 100, success: true},
				{durationMs: 200, success: true},
				{durationMs: 300, success: false},
				{durationMs: 400, success: true},
				{durationMs: 1000, success: false},
			},
			want: types.RunLatencyStats{
				SuccessRate: 0.6,
				P50Ms:       300,  // ceil(0.5*5)=3 → index 2
				P95Ms:       1000, // ceil(0.95*5)=5 → index 4
				Total:       5,
			},
		},
		{
			name: "all failures still compute duration percentiles",
			outcomes: []runLatencyOutcome{
				{durationMs: 50, success: false},
				{durationMs: 150, success: false},
			},
			want: types.RunLatencyStats{
				SuccessRate: 0,
				P50Ms:       50,  // ceil(0.5*2)=1 → index 0
				P95Ms:       150, // ceil(0.95*2)=2 → index 1
				Total:       2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := computeRunLatencyStats(tt.outcomes)
			assert.Equal(t, tt.want.Total, got.Total)
			assert.InDelta(t, tt.want.SuccessRate, got.SuccessRate, 0.001)
			assert.Equal(t, tt.want.P50Ms, got.P50Ms)
			assert.Equal(t, tt.want.P95Ms, got.P95Ms)
		})
	}
}

func TestPercentileNearestRank(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		sorted []int64
		p      float64
		want   int64
	}{
		{name: "empty slice", sorted: nil, p: 0.5, want: 0},
		{name: "p50 of four values", sorted: []int64{10, 20, 30, 40}, p: 0.5, want: 20},
		{name: "p95 of twenty values is 19th", sorted: []int64{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
			11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		}, p: 0.95, want: 19},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, percentileNearestRank(tt.sorted, tt.p))
		})
	}
}
