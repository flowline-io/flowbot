// Package store provides database storage implementations.
package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/llmusagerecord"
	"github.com/flowline-io/flowbot/pkg/types"
)

type LLMUsageStore struct {
	client *gen.Client
}

// NewLLMUsageStore returns a store backed by the given ent client.
func NewLLMUsageStore(client *gen.Client) *LLMUsageStore {
	return &LLMUsageStore{client: client}
}

// NewLLMUsageStoreFromDatabase returns a store using the global database client.
func NewLLMUsageStoreFromDatabase() *LLMUsageStore {
	client := ClientFromDB()
	if client == nil {
		return nil
	}
	return NewLLMUsageStore(client)
}

// RecordLLMUsage inserts one LLM usage row.
func (s *LLMUsageStore) RecordLLMUsage(ctx context.Context, record *types.LLMUsageRecordInput) error {
	if s == nil || s.client == nil {
		return errors.New("store: llm usage store unavailable")
	}
	if record == nil {
		return errors.New("store: nil llm usage record")
	}
	if strings.TrimSpace(record.UID) == "" {
		return errors.New("store: llm usage uid required")
	}
	source := strings.TrimSpace(record.Source)
	if source == "" {
		source = types.TokenUsageSourceAgent
	}
	source = types.NormalizeTokenUsageSource(source)
	_, err := s.client.LLMUsageRecord.Create().
		SetUID(record.UID).
		SetSessionID(record.SessionID).
		SetModel(record.Model).
		SetPromptTokens(record.PromptTokens).
		SetCompletionTokens(record.CompletionTokens).
		SetTotalTokens(record.TotalTokens).
		SetCacheRead(record.CacheRead).
		SetCacheWrite(record.CacheWrite).
		SetSource(source).
		SetCreatedAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("store: record llm usage: %w", err)
	}
	return nil
}

// TokenUsageStats aggregates usage for charts filtered by user and time range.
func (s *LLMUsageStore) TokenUsageStats(ctx context.Context, uid string, since, until time.Time, groupBy string) (*types.TokenUsageStats, error) {
	if s == nil || s.client == nil {
		return emptyTokenUsageStats(groupBy, since, until), nil
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return emptyTokenUsageStats(groupBy, since, until), nil
	}

	rows, err := s.client.LLMUsageRecord.Query().
		Where(
			llmusagerecord.UIDEQ(uid),
			llmusagerecord.CreatedAtGTE(since),
			llmusagerecord.CreatedAtLTE(until),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: list llm usage: %w", err)
	}

	stats := emptyTokenUsageStats(groupBy, since, until)
	for _, row := range rows {
		stats.Summary.PromptTokens += int64(row.PromptTokens)
		stats.Summary.CompletionTokens += int64(row.CompletionTokens)
		stats.Summary.TotalTokens += int64(row.TotalTokens)
	}

	dailyBuckets := buildTokenUsageDailyBuckets(rows, groupBy)
	stats.Series = buildTokenUsageSeries(dailyBuckets, stats.PeriodStart, stats.PeriodEnd, groupBy)
	return stats, nil
}

func emptyTokenUsageStats(groupBy string, since, until time.Time) *types.TokenUsageStats {
	startDay := startOfUTCDayStore(since)
	endDay := startOfUTCDayStore(until)
	return &types.TokenUsageStats{
		Summary:     types.TokenUsageSummary{},
		Series:      []types.TokenUsageSeries{},
		PeriodStart: startDay.Format("2006-01-02"),
		PeriodEnd:   endDay.Format("2006-01-02"),
		Today:       time.Now().UTC().Format("2006-01-02"),
		GroupBy:     groupBy,
	}
}

func startOfUTCDayStore(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func buildTokenUsageDailyBuckets(rows []*gen.LLMUsageRecord, groupBy string) map[string]map[string]int64 {
	buckets := make(map[string]map[string]int64)
	add := func(label, day string, amount int64) {
		if buckets[label] == nil {
			buckets[label] = make(map[string]int64)
		}
		buckets[label][day] += amount
	}

	for _, row := range rows {
		day := startOfUTCDayStore(row.CreatedAt).Format("2006-01-02")
		switch groupBy {
		case "usage_type":
			source := types.NormalizeTokenUsageSource(row.Source)
			add(source, day, int64(row.TotalTokens))
		default:
			label := strings.TrimSpace(row.Model)
			if label == "" {
				label = "unknown"
			}
			add(label, day, int64(row.TotalTokens))
		}
	}
	return buckets
}

func buildTokenUsageSeries(buckets map[string]map[string]int64, periodStart, periodEnd, groupBy string) []types.TokenUsageSeries {
	if groupBy == "usage_type" {
		for _, source := range types.TokenUsageSourceOrder {
			if buckets[source] == nil {
				buckets[source] = make(map[string]int64)
			}
		}
	}

	if len(buckets) == 0 {
		return []types.TokenUsageSeries{}
	}

	start, err := time.ParseInLocation("2006-01-02", periodStart, time.UTC)
	if err != nil {
		return []types.TokenUsageSeries{}
	}
	end, err := time.ParseInLocation("2006-01-02", periodEnd, time.UTC)
	if err != nil {
		return []types.TokenUsageSeries{}
	}

	labels := make([]string, 0, len(buckets))
	if groupBy == "usage_type" {
		labels = append(labels, types.TokenUsageSourceOrder...)
	} else {
		for label := range buckets {
			labels = append(labels, label)
		}
		slices.Sort(labels)
	}

	series := make([]types.TokenUsageSeries, 0, len(labels))
	for _, label := range labels {
		points := make([]types.TokenUsagePoint, 0)
		var cumulative int64
		for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
			dayKey := day.Format("2006-01-02")
			daily := buckets[label][dayKey]
			cumulative += daily
			points = append(points, types.TokenUsagePoint{
				Date:       dayKey,
				Daily:      daily,
				Cumulative: cumulative,
			})
		}
		displayLabel := label
		if groupBy == "usage_type" {
			displayLabel = types.TokenUsageSourceLabel(label)
		}
		series = append(series, types.TokenUsageSeries{
			Label:  displayLabel,
			Points: points,
		})
	}
	return series
}
