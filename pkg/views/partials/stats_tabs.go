package partials

import (
	"net/url"
	"strconv"
)

// StatsTabState is the active time-range and group-by selection for run stats charts.
type StatsTabState struct {
	RangeDays int    // 0 = all, otherwise 30 or 90
	GroupBy   string // day | week | month
}

func statsRangeBtnClass(activeDays, btnDays int) string {
	if activeDays == btnDays {
		return "join-item btn btn-sm btn-active"
	}
	return "join-item btn btn-sm"
}

func statsGroupByBtnClass(active, btn string) string {
	if active == btn {
		return "join-item btn btn-sm btn-active"
	}
	return "join-item btn btn-sm"
}

// PipelineStatsURL builds an HTMX URL for pipeline stats with days + groupBy tabs.
func PipelineStatsURL(name string, days int, groupBy string) string {
	return buildRunStatsURL("pipelines", name, days, groupBy)
}

// WorkflowStatsURL builds an HTMX URL for workflow stats with days + groupBy tabs.
func WorkflowStatsURL(name string, days int, groupBy string) string {
	return buildRunStatsURL("workflows", name, days, groupBy)
}

// FunctionStatsURL builds an HTMX URL for function stats with days + groupBy tabs.
func FunctionStatsURL(name string, days int, groupBy string) string {
	return buildRunStatsURL("functions", name, days, groupBy)
}

func buildRunStatsURL(kind, name string, days int, groupBy string) string {
	u := "/service/web/" + kind + "/stats"
	if name != "" {
		u = "/service/web/" + kind + "/" + url.PathEscape(name) + "/stats"
	}
	q := url.Values{}
	q.Set("groupBy", groupBy)
	q.Set("days", strconv.Itoa(days))
	return u + "?" + q.Encode()
}
