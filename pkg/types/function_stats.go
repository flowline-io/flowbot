package types

// FunctionStats holds aggregated function run statistics for chart rendering.
type FunctionStats struct {
	Summary              FunctionStatsSummary `json:"summary"`
	SuccessRateTrend     []SuccessRatePoint   `json:"success_rate_trend"`
	DurationDistribution []DurationEntry      `json:"duration_distribution"`
	VersionPie           []VersionRunCount    `json:"version_pie"`
}

// FunctionStatsSummary holds headline counters for the functions overview.
type FunctionStatsSummary struct {
	TotalFunctions int64 `json:"total_functions"`
	SuccessfulRuns int64 `json:"successful_runs"`
	FailedRuns     int64 `json:"failed_runs"`
}

// VersionRunCount counts function runs grouped by published version label (e.g. "v1").
type VersionRunCount struct {
	Version string `json:"version"`
	Count   int64  `json:"count"`
}
