package types

// RunLatencyStats holds per-definition run success rate and duration percentiles.
type RunLatencyStats struct {
	// SuccessRate is successful_runs / total completed runs in [0, 1].
	SuccessRate float64 `json:"success_rate"`
	// P50Ms is the 50th percentile of completed run durations in milliseconds.
	P50Ms int64 `json:"p50_ms"`
	// P95Ms is the 95th percentile of completed run durations in milliseconds.
	P95Ms int64 `json:"p95_ms"`
	// Total is the number of completed runs included in the aggregate.
	Total int64 `json:"total"`
}
