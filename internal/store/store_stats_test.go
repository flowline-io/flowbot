package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	_ "github.com/flowline-io/flowbot/internal/store/ent/gen/runtime"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

func TestPipelineStats_SuccessRateTrend(t *testing.T) {
	client := getTestClient(t)
	s := NewPipelineStore(client)
	ctx := context.Background()
	now := time.Now()

	sources := []pipelinerun.TriggerSource{
		pipelinerun.TriggerSourceEvent,
		pipelinerun.TriggerSourceEvent,
		pipelinerun.TriggerSourceWebhook,
		pipelinerun.TriggerSourceCron,
	}
	statuses := []int{int(schema.PipelineDone), int(schema.PipelineDone), int(schema.PipelineCancel), int(schema.PipelineDone)}
	for i, src := range sources {
		run, err := client.PipelineRun.Create().
			SetPipelineName("s1").
			SetEventID(fmt.Sprintf("eid-%s-%d", src, i)).
			SetEventType("t.evt").
			SetTriggerSource(src).
			SetStatus(int(schema.PipelineStart)).
			SetStartedAt(now).
			SetCreatedAt(now).
			Save(ctx)
		require.NoError(t, err)
		_, err = client.PipelineRun.UpdateOneID(run.ID).
			SetStatus(statuses[i]).
			SetCompletedAt(now).
			Save(ctx)
		require.NoError(t, err)
	}

	tests := []struct {
		name    string
		pName   string
		since   time.Time
		groupBy string
		minRows int
	}{
		{name: "global stats no time filter", pName: "", since: time.Time{}, groupBy: "day", minRows: 1},
		{name: "single pipeline", pName: "s1", since: time.Time{}, groupBy: "day", minRows: 1},
		{name: "future since returns empty", pName: "s1", since: now.Add(24 * time.Hour), groupBy: "day", minRows: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := s.PipelineStats(ctx, tt.pName, tt.since, tt.groupBy)
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.GreaterOrEqual(t, len(stats.SuccessRateTrend), tt.minRows)
			assert.Len(t, stats.TriggerSourcePie, 4)
			srcCount := make(map[string]bool)
			for _, sc := range stats.TriggerSourcePie {
				srcCount[sc.Source] = true
			}
			assert.True(t, srcCount["event"])
			assert.True(t, srcCount["webhook"])
			assert.True(t, srcCount["cron"])
			assert.True(t, srcCount["manual"])
			assert.Len(t, stats.DurationDistribution.Pipeline, 4)
			assert.Len(t, stats.DurationDistribution.Step, 4)
		})
	}
}

func TestPipelineStats_CompoundPipelineName(t *testing.T) {
	client := getTestClient(t)
	s := NewPipelineStore(client)
	ctx := context.Background()
	now := time.Now()

	run, err := client.PipelineRun.Create().
		SetPipelineName("test3__trigger_event_0").
		SetEventID("compound-event-1").
		SetEventType("t.evt").
		SetTriggerSource(pipelinerun.TriggerSourceEvent).
		SetStatus(int(schema.PipelineStart)).
		SetStartedAt(now.Add(-2 * time.Second)).
		SetCreatedAt(now).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.PipelineRun.UpdateOneID(run.ID).
		SetStatus(int(schema.PipelineDone)).
		SetCompletedAt(now).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name  string
		pName string
	}{
		{name: "parent name matches compound engine name", pName: "test3"},
		{name: "exact compound name still works", pName: "test3__trigger_event_0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := s.PipelineStats(ctx, tt.pName, time.Time{}, "day")
			require.NoError(t, err)
			require.NotNil(t, stats)
			require.Len(t, stats.SuccessRateTrend, 1)
			assert.Equal(t, int64(1), stats.SuccessRateTrend[0].Total)
			assert.Equal(t, int64(1), stats.SuccessRateTrend[0].Success)
			assert.InDelta(t, 1.0, stats.SuccessRateTrend[0].Rate, 0.001)
			assert.Equal(t, int64(1), stats.DurationDistribution.Pipeline[1].Count)
			assert.Equal(t, int64(1), stats.TriggerSourcePie[0].Count)
		})
	}
}

func TestPipelineStats_Summary(t *testing.T) {
	client := getTestClient(t)
	s := NewPipelineStore(client)
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, s.CreateDefinition(ctx, "summary-a", "", ""))
	require.NoError(t, s.CreateDefinition(ctx, "summary-b", "", ""))

	statuses := []int{
		int(schema.PipelineDone),
		int(schema.PipelineDone),
		int(schema.PipelineFailed),
		int(schema.PipelineCancel),
	}
	for i, status := range statuses {
		run, err := client.PipelineRun.Create().
			SetPipelineName("summary-a__trigger_event_0").
			SetEventID(fmt.Sprintf("summary-event-%d", i)).
			SetEventType("t.evt").
			SetTriggerSource(pipelinerun.TriggerSourceEvent).
			SetStatus(int(schema.PipelineStart)).
			SetStartedAt(now.Add(-time.Duration(i) * time.Hour)).
			SetCreatedAt(now).
			Save(ctx)
		require.NoError(t, err)
		_, err = client.PipelineRun.UpdateOneID(run.ID).
			SetStatus(status).
			SetCompletedAt(now).
			Save(ctx)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		pName          string
		since          time.Time
		wantPipelines  int64
		wantSuccessful int64
		wantFailed     int64
	}{
		{
			name:           "global summary counts definitions and completed outcomes",
			pName:          "",
			wantPipelines:  2,
			wantSuccessful: 2,
			wantFailed:     1,
		},
		{
			name:           "single pipeline summary uses parent name matching",
			pName:          "summary-a",
			wantPipelines:  0,
			wantSuccessful: 2,
			wantFailed:     1,
		},
		{
			name:           "since filter excludes older runs",
			pName:          "",
			since:          now.Add(-30 * time.Minute),
			wantPipelines:  2,
			wantSuccessful: 1,
			wantFailed:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := s.PipelineStats(ctx, tt.pName, tt.since, "day")
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, tt.wantPipelines, stats.Summary.TotalPipelines)
			assert.Equal(t, tt.wantSuccessful, stats.Summary.SuccessfulRuns)
			assert.Equal(t, tt.wantFailed, stats.Summary.FailedRuns)
		})
	}
}

func TestPipelineStats_NilSafe(t *testing.T) {
	tests := []struct {
		name  string
		store *PipelineStore
	}{
		{name: "nil store pointer", store: nil},
		{name: "zero-value store with nil client", store: &PipelineStore{}},
		{name: "zero-value store with explicit nil client", store: &PipelineStore{client: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := tt.store.PipelineStats(context.Background(), "", time.Time{}, "day")
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Len(t, stats.TriggerSourcePie, 4)
		})
	}
}
