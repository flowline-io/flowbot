package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestWorkflowStats_SuccessRateTrend(t *testing.T) {
	client := getTestClient(t)
	s := NewWorkflowStore(client)
	rs := NewWorkflowRunStore(client)
	ctx := context.Background()
	now := time.Now()

	_, err := s.ApplyDefinition(ctx, &types.WorkflowMetadata{
		Name: "wf-stats-trend", Enabled: true, Pipeline: []string{"a"},
		Tasks: []types.WorkflowTask{{ID: "a", Action: "shell:echo"}},
	})
	require.NoError(t, err)

	triggers := []string{"cron", "cron", "webhook", "manual"}
	statuses := []int{int(schema.WorkflowRunDone), int(schema.WorkflowRunDone), int(schema.WorkflowRunFailed), int(schema.WorkflowRunDone)}
	for i, trigger := range triggers {
		run, err := rs.CreateRun(ctx, 0, "wf-stats-trend", "db", trigger, nil, nil)
		require.NoError(t, err)
		require.NoError(t, client.WorkflowRun.UpdateOneID(run.ID).
			SetStatus(statuses[i]).
			SetStartedAt(now.Add(-2*time.Second)).
			SetCompletedAt(now).
			Exec(ctx))
	}

	tests := []struct {
		name    string
		wfName  string
		since   time.Time
		groupBy string
		minRows int
	}{
		{name: "global stats no time filter", wfName: "", since: time.Time{}, groupBy: "day", minRows: 1},
		{name: "single workflow", wfName: "wf-stats-trend", since: time.Time{}, groupBy: "day", minRows: 1},
		{name: "future since returns empty", wfName: "wf-stats-trend", since: now.Add(24 * time.Hour), groupBy: "day", minRows: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := s.WorkflowStats(ctx, tt.wfName, tt.since, tt.groupBy)
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.GreaterOrEqual(t, len(stats.SuccessRateTrend), tt.minRows)
			assert.Len(t, stats.TriggerSourcePie, 3)
			srcCount := make(map[string]bool)
			for _, sc := range stats.TriggerSourcePie {
				srcCount[sc.Source] = true
			}
			assert.True(t, srcCount["cron"])
			assert.True(t, srcCount["webhook"])
			assert.True(t, srcCount["manual"])
			assert.Len(t, stats.DurationDistribution.Workflow, 4)
		})
	}
}

func TestWorkflowStats_Summary(t *testing.T) {
	client := getTestClient(t)
	s := NewWorkflowStore(client)
	rs := NewWorkflowRunStore(client)
	ctx := context.Background()
	now := time.Now()

	_, err := s.ApplyDefinition(ctx, &types.WorkflowMetadata{
		Name: "summary-wf-a", Enabled: true, Pipeline: []string{"a"},
		Tasks: []types.WorkflowTask{{ID: "a", Action: "shell:echo"}},
	})
	require.NoError(t, err)
	_, err = s.ApplyDefinition(ctx, &types.WorkflowMetadata{
		Name: "summary-wf-b", Enabled: true, Pipeline: []string{"a"},
		Tasks: []types.WorkflowTask{{ID: "a", Action: "shell:echo"}},
	})
	require.NoError(t, err)

	statuses := []int{
		int(schema.WorkflowRunDone),
		int(schema.WorkflowRunDone),
		int(schema.WorkflowRunFailed),
	}
	for i, status := range statuses {
		run, err := rs.CreateRun(ctx, 0, "summary-wf-a", "db", "manual", nil, nil)
		require.NoError(t, err)
		require.NoError(t, client.WorkflowRun.UpdateOneID(run.ID).
			SetStatus(status).
			SetStartedAt(now.Add(-time.Duration(i)*time.Hour)).
			SetCompletedAt(now).
			Exec(ctx))
	}

	tests := []struct {
		name           string
		wfName         string
		since          time.Time
		wantWorkflows  int64
		wantSuccessful int64
		wantFailed     int64
	}{
		{
			name:           "global summary counts definitions and completed outcomes",
			wfName:         "",
			wantWorkflows:  2,
			wantSuccessful: 2,
			wantFailed:     1,
		},
		{
			name:           "single workflow summary omits total count",
			wfName:         "summary-wf-a",
			wantWorkflows:  0,
			wantSuccessful: 2,
			wantFailed:     1,
		},
		{
			name:           "since filter excludes older runs",
			wfName:         "",
			since:          now.Add(-30 * time.Minute),
			wantWorkflows:  2,
			wantSuccessful: 1,
			wantFailed:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := s.WorkflowStats(ctx, tt.wfName, tt.since, "day")
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, tt.wantWorkflows, stats.Summary.TotalWorkflows)
			assert.Equal(t, tt.wantSuccessful, stats.Summary.SuccessfulRuns)
			assert.Equal(t, tt.wantFailed, stats.Summary.FailedRuns)
		})
	}
}

func TestWorkflowStats_NilSafe(t *testing.T) {
	tests := []struct {
		name  string
		store *WorkflowStore
	}{
		{name: "nil store pointer", store: nil},
		{name: "zero-value store with nil client", store: &WorkflowStore{}},
		{name: "zero-value store with explicit nil client", store: &WorkflowStore{client: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := tt.store.WorkflowStats(context.Background(), "", time.Time{}, "day")
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Len(t, stats.TriggerSourcePie, 3)
		})
	}
}
