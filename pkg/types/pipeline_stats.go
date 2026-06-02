package types

// PipelineStats holds aggregated pipeline run statistics for chart rendering.
type PipelineStats struct {
	SuccessRateTrend     []SuccessRatePoint   `json:"success_rate_trend"`
	DurationDistribution DurationDistribution `json:"duration_distribution"`
	TriggerSourcePie     []TriggerSourceCount `json:"trigger_source_pie"`
}

// SuccessRatePoint is a single data point on the success rate trend chart.
type SuccessRatePoint struct {
	Date    string  `json:"date"`
	Total   int64   `json:"total"`
	Success int64   `json:"success"`
	Rate    float64 `json:"rate"`
}

// DurationDistribution holds pipeline and step duration bucket counts.
type DurationDistribution struct {
	Pipeline []DurationEntry `json:"pipeline"`
	Step     []DurationEntry `json:"step"`
}

// DurationEntry counts runs that fell into a named duration bucket.
type DurationEntry struct {
	Bucket string `json:"bucket"`
	Count  int64  `json:"count"`
}

// TriggerSourceCount counts pipeline runs grouped by trigger source.
type TriggerSourceCount struct {
	Source string `json:"source"`
	Count  int64  `json:"count"`
}
