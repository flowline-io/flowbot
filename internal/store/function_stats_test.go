package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/functionrun"
)

func TestFunctionStats_Summary(t *testing.T) {
	client := getTestClient(t)
	s := NewFunctionStore(client)
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, s.CreateDefinition(ctx, "fn-stats-a", ""))
	require.NoError(t, s.CreateDefinition(ctx, "fn-stats-b", ""))

	statuses := []functionrun.Status{
		functionrun.StatusSucceeded,
		functionrun.StatusSucceeded,
		functionrun.StatusFailed,
	}
	for i, status := range statuses {
		_, err := client.FunctionRun.Create().
			SetFunctionName("fn-stats-a").
			SetVersion(1).
			SetStatus(status).
			SetDurationMs(int64(100 * (i + 1))).
			SetCreatedAt(now.Add(-time.Duration(i) * time.Hour)).
			Save(ctx)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		fnName         string
		since          time.Time
		wantFunctions  int64
		wantSuccessful int64
		wantFailed     int64
	}{
		{
			name:           "global summary counts definitions and outcomes",
			fnName:         "",
			wantFunctions:  2,
			wantSuccessful: 2,
			wantFailed:     1,
		},
		{
			name:           "single function summary omits total count",
			fnName:         "fn-stats-a",
			wantFunctions:  0,
			wantSuccessful: 2,
			wantFailed:     1,
		},
		{
			name:           "since filter excludes older runs",
			fnName:         "",
			since:          now.Add(-30 * time.Minute),
			wantFunctions:  2,
			wantSuccessful: 1,
			wantFailed:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := s.FunctionStats(ctx, tt.fnName, tt.since, "day")
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, tt.wantFunctions, stats.Summary.TotalFunctions)
			assert.Equal(t, tt.wantSuccessful, stats.Summary.SuccessfulRuns)
			assert.Equal(t, tt.wantFailed, stats.Summary.FailedRuns)
			assert.Len(t, stats.DurationDistribution, 4)
		})
	}
}

func TestFunctionStats_NilSafe(t *testing.T) {
	tests := []struct {
		name  string
		store *FunctionStore
	}{
		{name: "nil store pointer", store: nil},
		{name: "zero-value store with nil client", store: &FunctionStore{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := tt.store.FunctionStats(context.Background(), "", time.Time{}, "day")
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Empty(t, stats.SuccessRateTrend)
			assert.Len(t, stats.DurationDistribution, 4)
		})
	}
}
