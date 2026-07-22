package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
	pkgworkflow "github.com/flowline-io/flowbot/pkg/workflow"
)

func TestWorkflowStore_ApplyGetDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, ws *store.WorkflowStore, rs *store.WorkflowRunStore)
	}{
		{
			name: "apply then get definition by name",
			run: func(t *testing.T, ws *store.WorkflowStore, _ *store.WorkflowRunStore) {
				ctx := context.Background()
				meta := sampleWorkflowMeta("apply-get-wf")
				row, err := ws.ApplyDefinition(ctx, meta)
				require.NoError(t, err)
				require.NotNil(t, row)
				assert.Equal(t, meta.Name, row.Name)
				assert.True(t, row.Enabled)

				dto, err := ws.GetDefinitionByName(ctx, meta.Name)
				require.NoError(t, err)
				require.NotNil(t, dto.Workflow)
				require.Len(t, dto.Tasks, 1)
				assert.Equal(t, "step1", dto.Tasks[0].TaskID)
				require.Len(t, dto.Triggers, 1)
				assert.Equal(t, "manual", dto.Triggers[0].Type)

				loaded, err := ws.GetMetadata(ctx, meta.Name)
				require.NoError(t, err)
				assert.Equal(t, meta.Name, loaded.Name)
				assert.Equal(t, meta.Pipeline, loaded.Pipeline)
				require.Len(t, loaded.Inputs, 1)
				assert.Equal(t, "url", loaded.Inputs[0].Name)
			},
		},
		{
			name: "apply upserts and replaces tasks",
			run: func(t *testing.T, ws *store.WorkflowStore, _ *store.WorkflowRunStore) {
				ctx := context.Background()
				meta := sampleWorkflowMeta("upsert-wf")
				_, err := ws.ApplyDefinition(ctx, meta)
				require.NoError(t, err)

				meta.Describe = "updated"
				meta.Tasks = []types.WorkflowTask{
					{ID: "step1", Action: "shell:echo", Params: types.KV{"msg": "a"}},
					{ID: "step2", Action: "mapper:", Params: types.KV{"k": "v"}},
				}
				meta.Pipeline = []string{"step1", "step2"}
				_, err = ws.ApplyDefinition(ctx, meta)
				require.NoError(t, err)

				dto, err := ws.GetDefinitionByName(ctx, meta.Name)
				require.NoError(t, err)
				assert.Equal(t, "updated", dto.Workflow.Describe)
				require.Len(t, dto.Tasks, 2)
			},
		},
		{
			name: "delete keeps runs and clears workflow_id",
			run: func(t *testing.T, ws *store.WorkflowStore, rs *store.WorkflowRunStore) {
				ctx := context.Background()
				meta := sampleWorkflowMeta("delete-keep-runs")
				row, err := ws.ApplyDefinition(ctx, meta)
				require.NoError(t, err)

				run, err := rs.CreateRun(ctx, row.ID, meta.Name, "db", "manual", nil, map[string]any{"url": "x"})
				require.NoError(t, err)
				require.NotNil(t, run.WorkflowID)
				assert.Equal(t, row.ID, *run.WorkflowID)

				err = ws.DeleteDefinitionByName(ctx, meta.Name)
				require.NoError(t, err)

				_, err = ws.GetDefinitionByName(ctx, meta.Name)
				require.ErrorIs(t, err, types.ErrNotFound)

				kept, err := rs.GetRun(ctx, run.ID)
				require.NoError(t, err)
				assert.Equal(t, meta.Name, kept.WorkflowName)
				assert.Nil(t, kept.WorkflowID)

				runs, err := ws.ListRunsByName(ctx, meta.Name)
				require.NoError(t, err)
				require.Len(t, runs, 1)
			},
		},
		{
			name: "set enabled updates flag",
			run: func(t *testing.T, ws *store.WorkflowStore, _ *store.WorkflowRunStore) {
				ctx := context.Background()
				meta := sampleWorkflowMeta("enable-wf")
				_, err := ws.ApplyDefinition(ctx, meta)
				require.NoError(t, err)

				row, err := ws.SetEnabled(ctx, meta.Name, false)
				require.NoError(t, err)
				assert.False(t, row.Enabled)

				list, err := ws.ListDefinitions(ctx)
				require.NoError(t, err)
				found := false
				for _, w := range list {
					if w.Name == meta.Name {
						found = true
						assert.False(t, w.Enabled)
					}
				}
				assert.True(t, found)
			},
		},
		{
			name: "set trigger enabled updates flag",
			run: func(t *testing.T, ws *store.WorkflowStore, _ *store.WorkflowRunStore) {
				ctx := context.Background()
				meta := sampleWorkflowMeta("trigger-enable-wf")
				meta.Triggers = []types.WorkflowTriggerDef{
					{Type: "manual", Enabled: true},
					{Type: "cron", Enabled: true, Rule: types.KV{"cron": "@hourly"}},
				}
				_, err := ws.ApplyDefinition(ctx, meta)
				require.NoError(t, err)

				dto, err := ws.GetDefinitionByName(ctx, meta.Name)
				require.NoError(t, err)
				require.Len(t, dto.Triggers, 2)
				cronID := dto.Triggers[1].ID

				row, err := ws.SetTriggerEnabled(ctx, meta.Name, cronID, false)
				require.NoError(t, err)
				assert.False(t, row.Enabled)
				assert.Equal(t, "cron", row.Type)

				dto, err = ws.GetDefinitionByName(ctx, meta.Name)
				require.NoError(t, err)
				assert.True(t, dto.Triggers[0].Enabled)
				assert.False(t, dto.Triggers[1].Enabled)
			},
		},
		{
			name: "set trigger enabled rejects foreign trigger",
			run: func(t *testing.T, ws *store.WorkflowStore, _ *store.WorkflowRunStore) {
				ctx := context.Background()
				metaA := sampleWorkflowMeta("trigger-owner-a")
				metaB := sampleWorkflowMeta("trigger-owner-b")
				_, err := ws.ApplyDefinition(ctx, metaA)
				require.NoError(t, err)
				_, err = ws.ApplyDefinition(ctx, metaB)
				require.NoError(t, err)

				dtoB, err := ws.GetDefinitionByName(ctx, metaB.Name)
				require.NoError(t, err)
				require.NotEmpty(t, dtoB.Triggers)

				_, err = ws.SetTriggerEnabled(ctx, metaA.Name, dtoB.Triggers[0].ID, false)
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := sqlitetest.OpenClient(t, t.Name())
			ws := store.NewWorkflowStore(client)
			rs := store.NewWorkflowRunStore(client)
			tt.run(t, ws, rs)
		})
	}
}

func sampleWorkflowMeta(name string) *types.WorkflowMetadata {
	return &types.WorkflowMetadata{
		Name:           name,
		Describe:       "sample",
		Enabled:        true,
		Resumable:      true,
		MaxConcurrency: 1,
		Inputs: []types.WorkflowInputDef{
			{Name: "url", Type: types.WorkflowInputTypeString, Required: true},
		},
		Triggers: []types.WorkflowTriggerDef{
			{Type: "manual", Enabled: true},
		},
		Pipeline: []string{"step1"},
		Tasks: []types.WorkflowTask{
			{
				ID:     "step1",
				Action: "shell:echo",
				Params: types.KV{"msg": "{{input.url}}"},
				Retry: &types.RetryConfig{
					MaxAttempts: 2,
					Backoff:     types.BackoffFixed,
				},
			},
		},
	}
}

func TestWorkflowStore_MetadataRoundtrip(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ws := store.NewWorkflowStore(client)
	ctx := context.Background()
	meta := sampleWorkflowMeta("roundtrip-wf")
	_, err := ws.ApplyDefinition(ctx, meta)
	require.NoError(t, err)

	dto, err := ws.GetDefinitionByName(ctx, meta.Name)
	require.NoError(t, err)
	got, err := pkgworkflow.MetadataFromRows(pkgworkflow.WorkflowRows{
		Workflow: dto.Workflow,
		Tasks:    dto.Tasks,
		Triggers: dto.Triggers,
	})
	require.NoError(t, err)
	assert.Equal(t, meta.Name, got.Name)
	assert.Equal(t, meta.Enabled, got.Enabled)
	require.Len(t, got.Tasks, 1)
	require.NotNil(t, got.Tasks[0].Retry)
	assert.Equal(t, 2, got.Tasks[0].Retry.MaxAttempts)
}

func TestWorkflowStore_LatestRunStartedAtByNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "empty names returns empty map",
			run: func(t *testing.T) {
				client := sqlitetest.OpenClient(t, t.Name())
				ws := store.NewWorkflowStore(client)
				got, err := ws.LatestRunStartedAtByNames(context.Background(), nil)
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		{
			name: "returns latest started_at per workflow name",
			run: func(t *testing.T) {
				ctx := context.Background()
				client := sqlitetest.OpenClient(t, t.Name())
				ws := store.NewWorkflowStore(client)
				rs := store.NewWorkflowRunStore(client)

				metaA := sampleWorkflowMeta("last-run-a")
				rowA, err := ws.ApplyDefinition(ctx, metaA)
				require.NoError(t, err)
				metaB := sampleWorkflowMeta("last-run-b")
				rowB, err := ws.ApplyDefinition(ctx, metaB)
				require.NoError(t, err)

				older := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
				newer := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
				newest := time.Date(2026, 1, 3, 15, 0, 0, 0, time.UTC)

				runA1, err := rs.CreateRun(ctx, rowA.ID, metaA.Name, "db", "manual", nil, nil)
				require.NoError(t, err)
				_, err = client.WorkflowRun.UpdateOneID(runA1.ID).SetStartedAt(older).Save(ctx)
				require.NoError(t, err)

				runA2, err := rs.CreateRun(ctx, rowA.ID, metaA.Name, "db", "manual", nil, nil)
				require.NoError(t, err)
				_, err = client.WorkflowRun.UpdateOneID(runA2.ID).SetStartedAt(newest).Save(ctx)
				require.NoError(t, err)

				runB1, err := rs.CreateRun(ctx, rowB.ID, metaB.Name, "db", "manual", nil, nil)
				require.NoError(t, err)
				_, err = client.WorkflowRun.UpdateOneID(runB1.ID).SetStartedAt(newer).Save(ctx)
				require.NoError(t, err)

				got, err := ws.LatestRunStartedAtByNames(ctx, []string{metaA.Name, metaB.Name, "never-run"})
				require.NoError(t, err)
				require.Contains(t, got, metaA.Name)
				require.Contains(t, got, metaB.Name)
				assert.True(t, got[metaA.Name].UTC().Equal(newest))
				assert.True(t, got[metaB.Name].UTC().Equal(newer))
				_, hasNever := got["never-run"]
				assert.False(t, hasNever)
			},
		},
		{
			name: "nil store returns empty map",
			run: func(t *testing.T) {
				var ws *store.WorkflowStore
				got, err := ws.LatestRunStartedAtByNames(context.Background(), []string{"x"})
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestWorkflowStore_RunLatencyStatsByNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "empty names returns empty map",
			run: func(t *testing.T) {
				client := sqlitetest.OpenClient(t, t.Name())
				ws := store.NewWorkflowStore(client)
				got, err := ws.RunLatencyStatsByNames(context.Background(), nil, time.Time{})
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		{
			name: "aggregates success rate and percentiles per workflow",
			run: func(t *testing.T) {
				ctx := context.Background()
				client := sqlitetest.OpenClient(t, t.Name())
				ws := store.NewWorkflowStore(client)
				rs := store.NewWorkflowRunStore(client)

				metaA := sampleWorkflowMeta("lat-wf-a")
				rowA, err := ws.ApplyDefinition(ctx, metaA)
				require.NoError(t, err)
				metaB := sampleWorkflowMeta("lat-wf-b")
				rowB, err := ws.ApplyDefinition(ctx, metaB)
				require.NoError(t, err)

				now := time.Now()
				seed := func(wfID int64, name string, status int, start, end time.Time) {
					t.Helper()
					run, err := rs.CreateRun(ctx, wfID, name, "db", "manual", nil, nil)
					require.NoError(t, err)
					_, err = client.WorkflowRun.UpdateOneID(run.ID).
						SetStatus(status).
						SetStartedAt(start).
						SetCompletedAt(end).
						Save(ctx)
					require.NoError(t, err)
				}
				seed(rowA.ID, metaA.Name, int(schema.WorkflowRunDone), now.Add(-1000*time.Millisecond), now)
				seed(rowA.ID, metaA.Name, int(schema.WorkflowRunFailed), now.Add(-3000*time.Millisecond), now)
				seed(rowB.ID, metaB.Name, int(schema.WorkflowRunDone), now.Add(-500*time.Millisecond), now)

				got, err := ws.RunLatencyStatsByNames(ctx, []string{metaA.Name, metaB.Name, "never-run"}, time.Time{})
				require.NoError(t, err)
				require.Contains(t, got, metaA.Name)
				require.Contains(t, got, metaB.Name)
				a := got[metaA.Name]
				assert.Equal(t, int64(2), a.Total)
				assert.InDelta(t, 0.5, a.SuccessRate, 0.001)
				assert.Equal(t, int64(1000), a.P50Ms)
				assert.Equal(t, int64(3000), a.P95Ms)
				b := got[metaB.Name]
				assert.Equal(t, int64(1), b.Total)
				assert.InDelta(t, 1.0, b.SuccessRate, 0.001)
				_, hasNever := got["never-run"]
				assert.False(t, hasNever)
			},
		},
		{
			name: "nil store returns empty map",
			run: func(t *testing.T) {
				var ws *store.WorkflowStore
				got, err := ws.RunLatencyStatsByNames(context.Background(), []string{"x"}, time.Time{})
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestWorkflowRunStore_GetStepRunsByRunID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "empty run returns empty slice",
			run: func(t *testing.T) {
				client := sqlitetest.OpenClient(t, t.Name())
				rs := store.NewWorkflowRunStore(client)
				got, err := rs.GetStepRunsByRunID(context.Background(), 1)
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		{
			name: "returns steps ordered by id",
			run: func(t *testing.T) {
				ctx := context.Background()
				client := sqlitetest.OpenClient(t, t.Name())
				ws := store.NewWorkflowStore(client)
				rs := store.NewWorkflowRunStore(client)
				meta := sampleWorkflowMeta("step-runs-wf")
				row, err := ws.ApplyDefinition(ctx, meta)
				require.NoError(t, err)
				run, err := rs.CreateRun(ctx, row.ID, meta.Name, "db", "manual", nil, nil)
				require.NoError(t, err)

				_, err = rs.CreateStepRun(ctx, run.ID, "step1", "step1", "mapper:", "mapper", map[string]any{"a": 1}, 1)
				require.NoError(t, err)
				_, err = rs.CreateStepRun(ctx, run.ID, "step2", "step2", "shell:echo", "shell", map[string]any{"msg": "hi"}, 1)
				require.NoError(t, err)

				got, err := rs.GetStepRunsByRunID(ctx, run.ID)
				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.Equal(t, "step1", got[0].StepID)
				assert.Equal(t, "step2", got[1].StepID)
				assert.Equal(t, "mapper:", got[0].Action)
			},
		},
		{
			name: "nil store returns nil",
			run: func(t *testing.T) {
				var rs *store.WorkflowRunStore
				got, err := rs.GetStepRunsByRunID(context.Background(), 1)
				require.NoError(t, err)
				assert.Nil(t, got)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
