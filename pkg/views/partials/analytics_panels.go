package partials

import (
	"fmt"
	"net/url"
)

func analyticsEventsURL(activeRange, groupBy string) string {
	q := url.Values{}
	if activeRange == "" || activeRange == "custom" {
		activeRange = "7d"
	}
	q.Set("range", activeRange)
	q.Set("groupBy", groupBy)
	return "/service/web/analytics/events/panel?" + q.Encode()
}

func analyticsActivityURL(activeRange string) string {
	q := url.Values{}
	if activeRange == "" || activeRange == "custom" {
		activeRange = "7d"
	}
	q.Set("range", activeRange)
	return "/service/web/analytics/activity/panel?" + q.Encode()
}

func analyticsOpsURL(activeRange, groupBy string) string {
	q := url.Values{}
	if activeRange == "" || activeRange == "custom" {
		activeRange = "7d"
	}
	if groupBy == "" {
		groupBy = "day"
	}
	q.Set("range", activeRange)
	q.Set("groupBy", groupBy)
	return "/service/web/analytics/ops/runs/panel?" + q.Encode()
}

func analyticsRangeBtnClass(activeRange, preset string) string {
	if activeRange == preset {
		return "join-item btn btn-sm btn-active"
	}
	return "join-item btn btn-sm"
}

func formatAnalyticsInt(n int64) string {
	return fmt.Sprint(n)
}

func formatAnalyticsRate(rate float64) string {
	return fmt.Sprintf("%.0f%%", rate*100)
}

func formatAnalyticsDurationBucket(bucket string, count int64) string {
	return fmt.Sprintf("%s: %d", bucket, count)
}
