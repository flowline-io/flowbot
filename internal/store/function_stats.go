package store

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/functionrun"
	"github.com/flowline-io/flowbot/pkg/types"
)

// FunctionStats returns aggregated function run statistics for chart rendering.
// name empty = all functions. since zero = no time filter. groupBy = "day"|"week"|"month".
func (s *FunctionStore) FunctionStats(ctx context.Context, name string, since time.Time, groupBy string) (*types.FunctionStats, error) {
	if s == nil || s.client == nil {
		return emptyFunctionStats(), nil
	}
	stats := &types.FunctionStats{}

	var err error
	stats.Summary, err = s.loadFunctionStatsSummary(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("summary: %w", err)
	}

	runs, err := s.fetchTerminalFunctionRuns(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("terminal runs: %w", err)
	}
	stats.SuccessRateTrend = computeFunctionSuccessRate(runs, groupBy)
	stats.DurationDistribution = computeFunctionDurationBuckets(runs)
	stats.VersionPie = computeFunctionVersionPie(runs)
	return stats, nil
}

func (s *FunctionStore) loadFunctionStatsSummary(ctx context.Context, name string, since time.Time) (types.FunctionStatsSummary, error) {
	summary := types.FunctionStatsSummary{}
	if name == "" {
		count, err := s.client.FunctionDefinition.Query().Count(ctx)
		if err != nil {
			return summary, err
		}
		summary.TotalFunctions = int64(count)
	}

	successful, err := s.countFunctionRunsByStatus(ctx, name, since, functionrun.StatusSucceeded)
	if err != nil {
		return summary, err
	}
	failed, err := s.countFunctionRunsByStatus(ctx, name, since, functionrun.StatusFailed)
	if err != nil {
		return summary, err
	}
	summary.SuccessfulRuns = successful
	summary.FailedRuns = failed
	return summary, nil
}

func (s *FunctionStore) countFunctionRunsByStatus(ctx context.Context, name string, since time.Time, status functionrun.Status) (int64, error) {
	q := s.client.FunctionRun.Query().Where(functionrun.StatusEQ(status))
	if name != "" {
		q = q.Where(functionrun.FunctionNameEQ(name))
	}
	if !since.IsZero() {
		q = q.Where(functionrun.CreatedAtGTE(since))
	}
	count, err := q.Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (s *FunctionStore) fetchTerminalFunctionRuns(ctx context.Context, name string, since time.Time) ([]*gen.FunctionRun, error) {
	q := s.client.FunctionRun.Query().Where(
		functionrun.StatusIn(functionrun.StatusSucceeded, functionrun.StatusFailed),
	)
	if name != "" {
		q = q.Where(functionrun.FunctionNameEQ(name))
	}
	if !since.IsZero() {
		q = q.Where(functionrun.CreatedAtGTE(since))
	}
	return q.All(ctx)
}

func computeFunctionSuccessRate(runs []*gen.FunctionRun, groupBy string) []types.SuccessRatePoint {
	type dayStats struct {
		total   int64
		success int64
	}
	buckets := make(map[string]*dayStats)
	for _, r := range runs {
		if r == nil {
			continue
		}
		key := dateGroupKey(r.CreatedAt, groupBy)
		if buckets[key] == nil {
			buckets[key] = &dayStats{}
		}
		buckets[key].total++
		if r.Status == functionrun.StatusSucceeded {
			buckets[key].success++
		}
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	points := make([]types.SuccessRatePoint, 0, len(keys))
	for _, k := range keys {
		s := buckets[k]
		rate := float64(0)
		if s.total > 0 {
			rate = float64(s.success) / float64(s.total)
		}
		points = append(points, types.SuccessRatePoint{
			Date: k, Total: s.total, Success: s.success, Rate: rate,
		})
	}
	if points == nil {
		points = []types.SuccessRatePoint{}
	}
	return points
}

func computeFunctionDurationBuckets(runs []*gen.FunctionRun) []types.DurationEntry {
	result := emptyDurationBuckets()
	for _, r := range runs {
		if r == nil || r.DurationMs <= 0 {
			continue
		}
		incrementDurationBucket(result, time.Duration(r.DurationMs)*time.Millisecond)
	}
	return result
}

func computeFunctionVersionPie(runs []*gen.FunctionRun) []types.VersionRunCount {
	counts := make(map[string]int64)
	for _, r := range runs {
		if r == nil {
			continue
		}
		key := fmt.Sprintf("v%d", r.Version)
		counts[key]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	out := make([]types.VersionRunCount, 0, len(keys))
	for _, k := range keys {
		out = append(out, types.VersionRunCount{Version: k, Count: counts[k]})
	}
	if out == nil {
		out = []types.VersionRunCount{}
	}
	return out
}

func emptyFunctionStats() *types.FunctionStats {
	return &types.FunctionStats{
		Summary:              types.FunctionStatsSummary{},
		SuccessRateTrend:     []types.SuccessRatePoint{},
		DurationDistribution: emptyDurationBuckets(),
		VersionPie:           []types.VersionRunCount{},
	}
}
