package store

import (
	"math"
	"slices"

	"github.com/flowline-io/flowbot/pkg/types"
)

// runLatencyOutcome is one completed run used to aggregate latency stats.
type runLatencyOutcome struct {
	durationMs int64
	success    bool
}

// computeRunLatencyStats aggregates success rate and duration percentiles.
func computeRunLatencyStats(outcomes []runLatencyOutcome) types.RunLatencyStats {
	if len(outcomes) == 0 {
		return types.RunLatencyStats{}
	}
	durations := make([]int64, 0, len(outcomes))
	var success int64
	for _, o := range outcomes {
		durations = append(durations, o.durationMs)
		if o.success {
			success++
		}
	}
	slices.Sort(durations)
	total := int64(len(outcomes))
	return types.RunLatencyStats{
		SuccessRate: float64(success) / float64(total),
		P50Ms:       percentileNearestRank(durations, 0.50),
		P95Ms:       percentileNearestRank(durations, 0.95),
		Total:       total,
	}
}

// percentileNearestRank returns the nearest-rank percentile from a sorted ascending slice.
func percentileNearestRank(sorted []int64, p float64) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	rank := max(int(math.Ceil(p*float64(n))), 1)
	if rank > n {
		rank = n
	}
	return sorted[rank-1]
}
