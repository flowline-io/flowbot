package types

// ActivityStats holds personal pipeline/function run counts for chart rendering.
// Workflow runs are omitted: definitions lack created_by ownership today.
type ActivityStats struct {
	Summary     ActivityStatsSummary  `json:"summary"`
	Series      []ActivityStatsSeries `json:"series"`
	PeriodStart string                `json:"period_start"`
	PeriodEnd   string                `json:"period_end"`
	RangeLabel  string                `json:"range_label"`
	ActiveRange string                `json:"active_range"`
}

// ActivityStatsSummary holds headline run counters for the selected period.
type ActivityStatsSummary struct {
	PipelineRuns int64 `json:"pipeline_runs"`
	FunctionRuns int64 `json:"function_runs"`
}

// ActivityStatsSeries is one chart line for a run kind.
type ActivityStatsSeries struct {
	Kind   string               `json:"kind"`
	Label  string               `json:"label"`
	Points []ActivityStatsPoint `json:"points"`
}

// ActivityStatsPoint is one day on an activity series chart.
type ActivityStatsPoint struct {
	Date  string `json:"date"`
	Daily int64  `json:"daily"`
}

// AnalyticsOverview holds headline counters for the analytics landing page.
type AnalyticsOverview struct {
	TokenTotal     int64  `json:"token_total"`
	EventTotal     int64  `json:"event_total"`
	PipelineRuns   int64  `json:"pipeline_runs"`
	FunctionRuns   int64  `json:"function_runs"`
	PeriodStart    string `json:"period_start"`
	PeriodEnd      string `json:"period_end"`
	RangeLabel     string `json:"range_label"`
	ActiveRange    string `json:"active_range"`
	OpsSuccessRate float64 `json:"ops_success_rate"`
	OpsFailedRuns  int64   `json:"ops_failed_runs"`
}
