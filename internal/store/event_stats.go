package store

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/pkg/types"
)

const eventStatsTopSeries = 8

// EventStats aggregates data_events for charts filtered by user and time range.
// groupBy must be event_type, capability, or source.
func (s *EventStore) EventStats(ctx context.Context, uid string, since, until time.Time, groupBy string) (*types.EventStats, error) {
	if s == nil || s.client == nil {
		return emptyEventStats(groupBy, since, until), nil
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return emptyEventStats(groupBy, since, until), nil
	}
	groupBy, err := types.NormalizeEventStatsGroupBy(groupBy)
	if err != nil {
		return nil, err
	}

	rows, err := s.client.DataEvent.Query().
		Where(
			dataevent.UIDEQ(uid),
			dataevent.CreatedAtGTE(since),
			dataevent.CreatedAtLTE(until),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list data events for stats: %w", err)
	}

	stats := emptyEventStats(groupBy, since, until)
	stats.Summary.TotalEvents = int64(len(rows))
	dailyBuckets := map[string]map[string]int64{}
	keyTotals := map[string]int64{}
	for _, row := range rows {
		key := eventStatsGroupKey(row, groupBy)
		if key == "" {
			key = "(empty)"
		}
		day := startOfUTCDayStore(row.CreatedAt).Format("2006-01-02")
		if dailyBuckets[key] == nil {
			dailyBuckets[key] = map[string]int64{}
		}
		dailyBuckets[key][day]++
		keyTotals[key]++
	}
	stats.Summary.DistinctKeys = int64(len(keyTotals))
	stats.Series = buildEventStatsSeries(dailyBuckets, keyTotals, stats.PeriodStart, stats.PeriodEnd)
	if len(stats.Series) > 0 {
		stats.Summary.TopKeyCount = keyTotals[stats.Series[0].Label]
	}
	return stats, nil
}

func eventStatsGroupKey(row *gen.DataEvent, groupBy string) string {
	switch groupBy {
	case "capability":
		return strings.TrimSpace(row.Capability)
	case "source":
		return strings.TrimSpace(row.Source)
	default:
		return strings.TrimSpace(row.EventType)
	}
}

func buildEventStatsSeries(
	dailyBuckets map[string]map[string]int64,
	keyTotals map[string]int64,
	periodStart, periodEnd string,
) []types.EventStatsSeries {
	keys := make([]string, 0, len(keyTotals))
	for k := range keyTotals {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if keyTotals[a] != keyTotals[b] {
			return int(keyTotals[b] - keyTotals[a])
		}
		return strings.Compare(a, b)
	})
	if len(keys) > eventStatsTopSeries {
		other := map[string]int64{}
		for _, k := range keys[eventStatsTopSeries:] {
			for day, n := range dailyBuckets[k] {
				other[day] += n
			}
		}
		keys = keys[:eventStatsTopSeries]
		if len(other) > 0 {
			dailyBuckets["(other)"] = other
			keys = append(keys, "(other)")
		}
	}

	startDay, _ := time.ParseInLocation("2006-01-02", periodStart, time.UTC)
	endDay, _ := time.ParseInLocation("2006-01-02", periodEnd, time.UTC)
	out := make([]types.EventStatsSeries, 0, len(keys))
	for _, key := range keys {
		var cum int64
		points := make([]types.EventStatsPoint, 0)
		for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
			day := d.Format("2006-01-02")
			daily := dailyBuckets[key][day]
			cum += daily
			points = append(points, types.EventStatsPoint{Date: day, Daily: daily, Cumulative: cum})
		}
		out = append(out, types.EventStatsSeries{Label: key, Points: points})
	}
	return out
}

func emptyEventStats(groupBy string, since, until time.Time) *types.EventStats {
	startDay := startOfUTCDayStore(since)
	endDay := startOfUTCDayStore(until)
	return &types.EventStats{
		Summary:     types.EventStatsSummary{},
		Series:      []types.EventStatsSeries{},
		PeriodStart: startDay.Format("2006-01-02"),
		PeriodEnd:   endDay.Format("2006-01-02"),
		GroupBy:     groupBy,
	}
}
