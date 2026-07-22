package partials

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types"
)

// FormatDurationMs formats a millisecond duration for list table cells.
func FormatDurationMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

// RunLatencySuccessOrDash formats success rate as a percent, or an em dash when unknown.
func RunLatencySuccessOrDash(s *types.RunLatencyStats) string {
	if s == nil || s.Total == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", s.SuccessRate*100)
}

// RunLatencyP50OrDash formats P50 duration, or an em dash when unknown.
func RunLatencyP50OrDash(s *types.RunLatencyStats) string {
	if s == nil || s.Total == 0 {
		return "—"
	}
	return FormatDurationMs(s.P50Ms)
}

// RunLatencyP95OrDash formats P95 duration, or an em dash when unknown.
func RunLatencyP95OrDash(s *types.RunLatencyStats) string {
	if s == nil || s.Total == 0 {
		return "—"
	}
	return FormatDurationMs(s.P95Ms)
}

// RunLatencyCompactOrDash formats success · P50 / P95 for a single list column.
func RunLatencyCompactOrDash(s *types.RunLatencyStats) string {
	if s == nil || s.Total == 0 {
		return "—"
	}
	return fmt.Sprintf("%s · %s / %s",
		RunLatencySuccessOrDash(s),
		FormatDurationMs(s.P50Ms),
		FormatDurationMs(s.P95Ms))
}

// RunLatencyCompactTip is the tooltip explaining the compact 7d runs column.
func RunLatencyCompactTip(s *types.RunLatencyStats) string {
	if s == nil || s.Total == 0 {
		return "No completed runs in the last 7 days"
	}
	return fmt.Sprintf("Last 7 days: %s success, P50 %s, P95 %s (%d runs)",
		RunLatencySuccessOrDash(s),
		FormatDurationMs(s.P50Ms),
		FormatDurationMs(s.P95Ms),
		s.Total)
}

// AttachRunLatencyStats copies matching per-pipeline latency stats onto list entries.
func AttachRunLatencyStats(entries []PipelineListEntry, stats map[string]types.RunLatencyStats) []PipelineListEntry {
	if len(entries) == 0 || len(stats) == 0 {
		return entries
	}
	for i := range entries {
		if entries[i].Definition == nil {
			continue
		}
		st, ok := stats[entries[i].Definition.Name]
		if !ok || st.Total == 0 {
			continue
		}
		stCopy := st
		entries[i].Stats = &stCopy
	}
	return entries
}

// AttachWorkflowRunLatencyStats copies matching per-workflow latency stats onto list entries.
func AttachWorkflowRunLatencyStats(entries []WorkflowListEntry, stats map[string]types.RunLatencyStats) []WorkflowListEntry {
	if len(entries) == 0 || len(stats) == 0 {
		return entries
	}
	for i := range entries {
		st, ok := stats[entries[i].Name]
		if !ok || st.Total == 0 {
			continue
		}
		stCopy := st
		entries[i].Stats = &stCopy
	}
	return entries
}
