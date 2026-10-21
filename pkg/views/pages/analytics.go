package pages

import (
	"fmt"
	"net/url"
)

func analyticsRangeBtnClass(activeRange, preset string) string {
	if activeRange == preset {
		return "join-item btn btn-sm btn-active"
	}
	return "join-item btn btn-sm"
}

func formatAnalyticsRate(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate)
}

func formatAnalyticsCount(n int64) string {
	return fmt.Sprint(n)
}

func analyticsURL(path, rangeQuery, extra string) string {
	u := path
	sep := "?"
	if rangeQuery != "" {
		u += sep + rangeQuery
		sep = "&"
	}
	if extra != "" {
		u += sep + extra
	}
	return u
}

func analyticsOverviewURL(activeRange string) string {
	q := url.Values{}
	if activeRange == "" {
		activeRange = "7d"
	}
	q.Set("range", activeRange)
	return "/service/web/analytics?" + q.Encode()
}
