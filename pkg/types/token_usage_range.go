package types

import (
	"fmt"
	"strings"
	"time"
)

const maxTokenUsageCustomDays = 366

// ResolveTokenUsageRange converts a preset or custom date range to UTC bounds.
func ResolveTokenUsageRange(rangePreset, sinceStr, untilStr string, now time.Time) (since, until time.Time, activeRange, rangeLabel string, err error) {
	now = now.UTC()
	sinceStr = strings.TrimSpace(sinceStr)
	untilStr = strings.TrimSpace(untilStr)

	if sinceStr != "" || untilStr != "" {
		return resolveCustomTokenUsageRange(sinceStr, untilStr)
	}

	preset := strings.TrimSpace(rangePreset)
	if preset == "" {
		preset = "7d"
	}
	since, until, err = resolvePresetTokenUsageRange(preset, now)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", err
	}
	endDay := startOfUTCDay(until)
	return since, until, preset, formatTokenUsageRangeLabel(since, endDay), nil
}

func resolveCustomTokenUsageRange(sinceStr, untilStr string) (since, until time.Time, activeRange, rangeLabel string, err error) {
	if sinceStr == "" || untilStr == "" {
		return time.Time{}, time.Time{}, "", "", fmt.Errorf("%w: since and until must both be set", ErrInvalidArgument)
	}
	since, err = time.ParseInLocation("2006-01-02", sinceStr, time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", fmt.Errorf("%w: invalid since date", ErrInvalidArgument)
	}
	untilDay, err := time.ParseInLocation("2006-01-02", untilStr, time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", fmt.Errorf("%w: invalid until date", ErrInvalidArgument)
	}
	if since.After(untilDay) {
		return time.Time{}, time.Time{}, "", "", fmt.Errorf("%w: since after until", ErrInvalidArgument)
	}
	until = untilDay.Add(24*time.Hour - time.Nanosecond)
	if int(untilDay.Sub(since).Hours()/24)+1 > maxTokenUsageCustomDays {
		return time.Time{}, time.Time{}, "", "", fmt.Errorf("%w: custom range exceeds %d days", ErrInvalidArgument, maxTokenUsageCustomDays)
	}
	return since, until, "custom", formatTokenUsageRangeLabel(since, untilDay), nil
}

func resolvePresetTokenUsageRange(preset string, now time.Time) (since, until time.Time, err error) {
	switch preset {
	case "1d":
		return startOfUTCDay(now), now, nil
	case "7d":
		return startOfUTCDay(now.AddDate(0, 0, -6)), now, nil
	case "30d":
		return startOfUTCDay(now.AddDate(0, 0, -29)), now, nil
	case "mtd":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), now, nil
	case "last_month":
		firstThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		lastMonthEnd := firstThisMonth.Add(-time.Nanosecond)
		since := time.Date(lastMonthEnd.Year(), lastMonthEnd.Month(), 1, 0, 0, 0, 0, time.UTC)
		return since, lastMonthEnd, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid range %q", ErrInvalidArgument, preset)
	}
}

func startOfUTCDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func formatTokenUsageRangeLabel(since, untilDay time.Time) string {
	return since.Format("Jan 02") + " - " + untilDay.Format("Jan 02")
}

// NormalizeTokenUsageGroupBy validates and defaults the groupBy query value.
func NormalizeTokenUsageGroupBy(groupBy string) (string, error) {
	groupBy = strings.TrimSpace(groupBy)
	if groupBy == "" {
		return "model", nil
	}
	if groupBy != "model" && groupBy != "usage_type" {
		return "", fmt.Errorf("%w: groupBy must be model or usage_type", ErrInvalidArgument)
	}
	return groupBy, nil
}
