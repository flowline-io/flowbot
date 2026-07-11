package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestLLMUsageStore_RecordAndStats(t *testing.T) {
	client := getTestClient(t)
	s := NewLLMUsageStore(client)
	ctx := context.Background()
	uid := "user-admin"
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	_, err := client.LLMUsageRecord.Create().
		SetUID(uid).
		SetSessionID("sess-1").
		SetModel("gpt-4o").
		SetSource(types.TokenUsageSourceAgent).
		SetPromptTokens(100).
		SetCompletionTokens(50).
		SetTotalTokens(150).
		SetCreatedAt(now.AddDate(0, 0, -2)).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.LLMUsageRecord.Create().
		SetUID(uid).
		SetSessionID("sess-1").
		SetModel("gpt-4o-mini").
		SetSource(types.TokenUsageSourcePipeline).
		SetPromptTokens(20).
		SetCompletionTokens(10).
		SetTotalTokens(30).
		SetCreatedAt(now).
		Save(ctx)
	require.NoError(t, err)

	since := now.AddDate(0, 0, -6)
	until := now

	tests := []struct {
		name       string
		groupBy    string
		wantSeries int
		wantTotal  int64
	}{
		{name: "group by model", groupBy: "model", wantSeries: 2, wantTotal: 180},
		{name: "group by usage type", groupBy: "usage_type", wantSeries: 4, wantTotal: 180},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := s.TokenUsageStats(ctx, uid, since, until, tt.groupBy)
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, tt.wantTotal, stats.Summary.TotalTokens)
			assert.Len(t, stats.Series, tt.wantSeries)
		})
	}
}

func TestLLMUsageStore_ZeroFillAndUnknownModel(t *testing.T) {
	client := getTestClient(t)
	s := NewLLMUsageStore(client)
	ctx := context.Background()
	uid := "user-test"
	day1 := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)

	_, err := client.LLMUsageRecord.Create().
		SetUID(uid).
		SetModel("").
		SetTotalTokens(10).
		SetPromptTokens(10).
		SetCreatedAt(day1).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.LLMUsageRecord.Create().
		SetUID(uid).
		SetModel("gpt-4o").
		SetTotalTokens(20).
		SetPromptTokens(20).
		SetCreatedAt(day3).
		Save(ctx)
	require.NoError(t, err)

	since := time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 7, 10, 23, 59, 59, 0, time.UTC)
	stats, err := s.TokenUsageStats(ctx, uid, since, until, "model")
	require.NoError(t, err)

	var unknownSeries, gptSeries *types.TokenUsageSeries
	for i := range stats.Series {
		switch stats.Series[i].Label {
		case "unknown":
			unknownSeries = &stats.Series[i]
		case "gpt-4o":
			gptSeries = &stats.Series[i]
		}
	}
	require.NotNil(t, unknownSeries)
	require.NotNil(t, gptSeries)
	require.Len(t, unknownSeries.Points, 3)
	require.Len(t, gptSeries.Points, 3)
	assert.Equal(t, int64(10), unknownSeries.Points[0].Daily)
	assert.Equal(t, int64(0), unknownSeries.Points[1].Daily)
	assert.Equal(t, int64(10), unknownSeries.Points[1].Cumulative)
	assert.Equal(t, int64(0), gptSeries.Points[1].Daily)
	assert.Equal(t, int64(20), gptSeries.Points[2].Daily)
}

func TestLLMUsageStore_UsageTypeLegacySources(t *testing.T) {
	client := getTestClient(t)
	s := NewLLMUsageStore(client)
	ctx := context.Background()
	uid := "user-admin"
	day := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

	_, err := client.LLMUsageRecord.Create().
		SetUID(uid).
		SetSessionID("sess-agent").
		SetModel("deepseek-v4-flash").
		SetSource("chat_agent").
		SetTotalTokens(100).
		SetPromptTokens(100).
		SetCreatedAt(day).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.LLMUsageRecord.Create().
		SetUID(uid).
		SetSessionID("sess-pipeline").
		SetModel("deepseek-v4-flash").
		SetSource("pipeline").
		SetTotalTokens(30).
		SetPromptTokens(30).
		SetCreatedAt(day).
		Save(ctx)
	require.NoError(t, err)

	since := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 7, 11, 23, 59, 59, 0, time.UTC)
	stats, err := s.TokenUsageStats(ctx, uid, since, until, "usage_type")
	require.NoError(t, err)
	require.Len(t, stats.Series, 4)

	var agentSeries, pipelineSeries *types.TokenUsageSeries
	for i := range stats.Series {
		switch stats.Series[i].Label {
		case "Agent":
			agentSeries = &stats.Series[i]
		case "Pipeline":
			pipelineSeries = &stats.Series[i]
		}
	}
	require.NotNil(t, agentSeries)
	require.NotNil(t, pipelineSeries)
	require.Len(t, agentSeries.Points, 1)
	require.Len(t, pipelineSeries.Points, 1)
	assert.Equal(t, int64(100), agentSeries.Points[0].Daily)
	assert.Equal(t, int64(30), pipelineSeries.Points[0].Daily)
	assert.Equal(t, int64(130), stats.Summary.TotalTokens)
}

func TestLLMUsageStore_UsageTypeEmptySeries(t *testing.T) {
	client := getTestClient(t)
	s := NewLLMUsageStore(client)
	ctx := context.Background()
	since := time.Now().UTC().AddDate(0, 0, -7)
	until := time.Now().UTC()

	stats, err := s.TokenUsageStats(ctx, "user-empty", since, until, "usage_type")
	require.NoError(t, err)
	require.Len(t, stats.Series, 4)
	assert.Equal(t, "Agent", stats.Series[0].Label)
	assert.Equal(t, "Pipeline", stats.Series[1].Label)
	assert.Equal(t, "Scheduled Task", stats.Series[2].Label)
	assert.Equal(t, "Subagent", stats.Series[3].Label)
}

func TestLLMUsageStore_EmptyAndNilSafe(t *testing.T) {
	ctx := context.Background()
	since := time.Now().UTC().AddDate(0, 0, -7)
	until := time.Now().UTC()

	tests := []struct {
		name  string
		store *LLMUsageStore
		uid   string
	}{
		{name: "nil store", store: nil, uid: "user-a"},
		{name: "empty uid", store: NewLLMUsageStore(getTestClient(t)), uid: ""},
		{name: "no rows", store: NewLLMUsageStore(getTestClient(t)), uid: "user-b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := tt.store.TokenUsageStats(ctx, tt.uid, since, until, "model")
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Empty(t, stats.Series)
			assert.Zero(t, stats.Summary.TotalTokens)
		})
	}
}

func TestLLMUsageStore_RecordValidation(t *testing.T) {
	ctx := context.Background()
	s := NewLLMUsageStore(getTestClient(t))

	tests := []struct {
		name   string
		store  *LLMUsageStore
		record *types.LLMUsageRecordInput
	}{
		{name: "nil store", store: nil, record: &types.LLMUsageRecordInput{UID: "u"}},
		{name: "nil record", store: s, record: nil},
		{name: "empty uid", store: s, record: &types.LLMUsageRecordInput{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.store.RecordLLMUsage(ctx, tt.record)
			require.Error(t, err)
		})
	}
}
