package types

// TokenUsageStats holds aggregated LLM token usage for chart rendering.
type TokenUsageStats struct {
	Summary     TokenUsageSummary  `json:"summary"`
	Series      []TokenUsageSeries `json:"series"`
	PeriodStart string             `json:"period_start"`
	PeriodEnd   string             `json:"period_end"`
	RangeLabel  string             `json:"range_label"`
	ActiveRange string             `json:"active_range"`
	Today       string             `json:"today"`
	GroupBy     string             `json:"group_by"`
}

// TokenUsageSummary holds headline counters for the selected period.
type TokenUsageSummary struct {
	TotalTokens      int64 `json:"total_tokens"`
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

// TokenUsageSeries is one chart line grouped by model or usage type.
type TokenUsageSeries struct {
	Label  string            `json:"label"`
	Points []TokenUsagePoint `json:"points"`
}

// TokenUsagePoint is one day on a usage series chart.
type TokenUsagePoint struct {
	Date       string `json:"date"`
	Daily      int64  `json:"daily"`
	Cumulative int64  `json:"cumulative"`
}

// LLMUsageRecordInput is the payload for persisting one LLM usage row.
type LLMUsageRecordInput struct {
	UID              string
	SessionID        string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CacheRead        int
	CacheWrite       int
	Source           string
}
