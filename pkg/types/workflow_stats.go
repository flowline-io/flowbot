package types

// WorkflowStats holds aggregated workflow run statistics for chart rendering.
type WorkflowStats struct {
	Summary              WorkflowStatsSummary           `json:"summary"`
	SuccessRateTrend     []SuccessRatePoint             `json:"success_rate_trend"`
	DurationDistribution WorkflowDurationDistribution   `json:"duration_distribution"`
	TriggerSourcePie     []TriggerSourceCount           `json:"trigger_source_pie"`
}

// WorkflowStatsSummary holds headline counters for the workflows overview.
type WorkflowStatsSummary struct {
	TotalWorkflows int64 `json:"total_workflows"`
	SuccessfulRuns int64 `json:"successful_runs"`
	FailedRuns     int64 `json:"failed_runs"`
}

// WorkflowDurationDistribution holds workflow run duration bucket counts.
type WorkflowDurationDistribution struct {
	Workflow []DurationEntry `json:"workflow"`
}
