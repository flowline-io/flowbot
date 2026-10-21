package types

import (
	"fmt"
	"strings"
)

// EventStats holds aggregated data_events counts for chart rendering.
type EventStats struct {
	Summary     EventStatsSummary  `json:"summary"`
	Series      []EventStatsSeries `json:"series"`
	PeriodStart string             `json:"period_start"`
	PeriodEnd   string             `json:"period_end"`
	RangeLabel  string             `json:"range_label"`
	ActiveRange string             `json:"active_range"`
	GroupBy     string             `json:"group_by"`
}

// EventStatsSummary holds headline counters for the selected period.
type EventStatsSummary struct {
	TotalEvents  int64 `json:"total_events"`
	DistinctKeys int64 `json:"distinct_keys"`
	TopKeyCount  int64 `json:"top_key_count"`
}

// EventStatsSeries is one chart line grouped by event_type, capability, or source.
type EventStatsSeries struct {
	Label  string            `json:"label"`
	Points []EventStatsPoint `json:"points"`
}

// EventStatsPoint is one day on an event series chart.
type EventStatsPoint struct {
	Date       string `json:"date"`
	Daily      int64  `json:"daily"`
	Cumulative int64  `json:"cumulative"`
}

// NormalizeEventStatsGroupBy validates and defaults the groupBy query value.
func NormalizeEventStatsGroupBy(groupBy string) (string, error) {
	groupBy = strings.TrimSpace(groupBy)
	if groupBy == "" {
		return "event_type", nil
	}
	switch groupBy {
	case "event_type", "capability", "source":
		return groupBy, nil
	default:
		return "", fmt.Errorf("%w: groupBy must be event_type, capability, or source", ErrInvalidArgument)
	}
}
