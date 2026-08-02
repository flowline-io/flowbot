// Package store provides database storage implementations.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflow"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowrun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowtask"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/workflowtrigger"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	pkgworkflow "github.com/flowline-io/flowbot/pkg/workflow"
)

// ---------------------------------------------------------------------------
// WorkflowStore
// ---------------------------------------------------------------------------

// WorkflowDefinitionDTO is a workflow definition with its tasks and triggers.
type WorkflowDefinitionDTO struct {
	Workflow *gen.Workflow
	Tasks    []*gen.WorkflowTask
	Triggers []*gen.WorkflowTrigger
}

// WorkflowStore persists normalized workflow definitions (tasks + triggers).
type WorkflowStore struct {
	client *gen.Client
}

// NewWorkflowStore creates a WorkflowStore backed by the given ent client.
func NewWorkflowStore(client *gen.Client) *WorkflowStore {
	return &WorkflowStore{client: client}
}

// WorkflowStoreFromDB returns a WorkflowStore using the global database client.
func WorkflowStoreFromDB() *WorkflowStore {
	return NewWorkflowStore(ClientFromDB())
}

// ApplyDefinition upserts a workflow definition by name and replaces all tasks and triggers.
func (s *WorkflowStore) ApplyDefinition(ctx context.Context, meta *types.WorkflowMetadata) (*gen.Workflow, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	if meta == nil {
		return nil, fmt.Errorf("workflow metadata is nil")
	}
	if err := schema.ValidatePipelineName(meta.Name); err != nil {
		return nil, fmt.Errorf("workflow name: %w", err)
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin workflow apply tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	row, err := upsertWorkflowRow(ctx, tx, meta)
	if err != nil {
		return nil, err
	}
	if err := createWorkflowTasks(ctx, tx, row.ID, meta.Tasks); err != nil {
		return nil, err
	}
	if err := createWorkflowTriggers(ctx, tx, row.ID, meta.Triggers); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit workflow apply: %w", err)
	}
	committed = true
	return row, nil
}

func upsertWorkflowRow(ctx context.Context, tx *gen.Tx, meta *types.WorkflowMetadata) (*gen.Workflow, error) {
	existing, err := tx.Workflow.Query().
		Where(workflow.NameEQ(meta.Name)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return nil, fmt.Errorf("lookup workflow %s: %w", meta.Name, err)
	}

	inputs := pkgworkflow.InputsToMaps(meta.Inputs)
	pipelineSteps := append([]string(nil), meta.Pipeline...)
	now := time.Now()

	if existing == nil {
		row, err := tx.Workflow.Create().
			SetName(meta.Name).
			SetDescribe(meta.Describe).
			SetEnabled(meta.Enabled).
			SetResumable(meta.Resumable).
			SetMaxConcurrency(meta.MaxConcurrency).
			SetInputs(inputs).
			SetPipeline(pipelineSteps).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("create workflow %s: %w", meta.Name, err)
		}
		return row, nil
	}

	row, err := tx.Workflow.UpdateOneID(existing.ID).
		SetDescribe(meta.Describe).
		SetEnabled(meta.Enabled).
		SetResumable(meta.Resumable).
		SetMaxConcurrency(meta.MaxConcurrency).
		SetInputs(inputs).
		SetPipeline(pipelineSteps).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update workflow %s: %w", meta.Name, err)
	}
	if _, err := tx.WorkflowTask.Delete().Where(workflowtask.WorkflowIDEQ(row.ID)).Exec(ctx); err != nil {
		return nil, fmt.Errorf("clear workflow tasks: %w", err)
	}
	if _, err := tx.WorkflowTrigger.Delete().Where(workflowtrigger.WorkflowIDEQ(row.ID)).Exec(ctx); err != nil {
		return nil, fmt.Errorf("clear workflow triggers: %w", err)
	}
	return row, nil
}

func createWorkflowTasks(ctx context.Context, tx *gen.Tx, workflowID int64, tasks []types.WorkflowTask) error {
	for _, t := range tasks {
		builder := tx.WorkflowTask.Create().
			SetWorkflowID(workflowID).
			SetTaskID(t.ID).
			SetAction(t.Action).
			SetDescribe(t.Describe).
			SetParams(map[string]any(t.Params)).
			SetVars(append([]string(nil), t.Vars...)).
			SetConn(append([]string(nil), t.Conn...))
		if retry := pkgworkflow.RetryToMap(t.Retry); retry != nil {
			builder = builder.SetRetry(retry)
		}
		if _, err := builder.Save(ctx); err != nil {
			return fmt.Errorf("create workflow task %s: %w", t.ID, err)
		}
	}
	return nil
}

func createWorkflowTriggers(ctx context.Context, tx *gen.Tx, workflowID int64, triggers []types.WorkflowTriggerDef) error {
	for _, tr := range triggers {
		if _, err := tx.WorkflowTrigger.Create().
			SetWorkflowID(workflowID).
			SetType(tr.Type).
			SetEnabled(tr.Enabled).
			SetRule(map[string]any(tr.Rule)).
			Save(ctx); err != nil {
			return fmt.Errorf("create workflow trigger %s: %w", tr.Type, err)
		}
	}
	return nil
}

// GetDefinitionByName returns a workflow definition with tasks and triggers.
func (s *WorkflowStore) GetDefinitionByName(ctx context.Context, name string) (*WorkflowDefinitionDTO, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	row, err := s.client.Workflow.Query().
		Where(workflow.NameEQ(name)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, fmt.Errorf("%w: workflow %s", types.ErrNotFound, name)
		}
		return nil, err
	}
	tasks, err := s.client.WorkflowTask.Query().
		Where(workflowtask.WorkflowIDEQ(row.ID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow tasks: %w", err)
	}
	triggers, err := s.client.WorkflowTrigger.Query().
		Where(workflowtrigger.WorkflowIDEQ(row.ID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow triggers: %w", err)
	}
	return &WorkflowDefinitionDTO{
		Workflow: row,
		Tasks:    tasks,
		Triggers: triggers,
	}, nil
}

// ListDefinitions returns all workflow definition rows (without tasks/triggers).
func (s *WorkflowStore) ListDefinitions(ctx context.Context) ([]*gen.Workflow, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.Workflow.Query().
		Order(gen.Asc(workflow.FieldName)).
		All(ctx)
}

// ListTriggers returns all workflow trigger rows.
func (s *WorkflowStore) ListTriggers(ctx context.Context) ([]*gen.WorkflowTrigger, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.WorkflowTrigger.Query().
		Order(gen.Asc(workflowtrigger.FieldID)).
		All(ctx)
}

// DeleteDefinitionByName deletes a workflow and its tasks/triggers.
// Existing runs are kept; their workflow_id is set to NULL.
func (s *WorkflowStore) DeleteDefinitionByName(ctx context.Context, name string) error {
	if s == nil || s.client == nil {
		return nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin workflow delete tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	row, err := tx.Workflow.Query().
		Where(workflow.NameEQ(name)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return fmt.Errorf("%w: workflow %s", types.ErrNotFound, name)
		}
		return err
	}

	if _, err := tx.WorkflowRun.Update().
		Where(workflowrun.WorkflowIDEQ(row.ID)).
		ClearWorkflowID().
		Save(ctx); err != nil {
		return fmt.Errorf("clear workflow_id on runs: %w", err)
	}
	if _, err := tx.WorkflowTask.Delete().Where(workflowtask.WorkflowIDEQ(row.ID)).Exec(ctx); err != nil {
		return fmt.Errorf("delete workflow tasks: %w", err)
	}
	if _, err := tx.WorkflowTrigger.Delete().Where(workflowtrigger.WorkflowIDEQ(row.ID)).Exec(ctx); err != nil {
		return fmt.Errorf("delete workflow triggers: %w", err)
	}
	if err := tx.Workflow.DeleteOneID(row.ID).Exec(ctx); err != nil {
		return fmt.Errorf("delete workflow %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit workflow delete: %w", err)
	}
	committed = true
	return nil
}

// SetEnabled updates the enabled flag for a workflow definition by name.
func (s *WorkflowStore) SetEnabled(ctx context.Context, name string, enabled bool) (*gen.Workflow, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	n, err := s.client.Workflow.Update().
		Where(workflow.NameEQ(name)).
		SetEnabled(enabled).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("%w: workflow %s", types.ErrNotFound, name)
	}
	return s.client.Workflow.Query().Where(workflow.NameEQ(name)).Only(ctx)
}

// SetTriggerEnabled updates the enabled flag for one trigger belonging to a named workflow.
func (s *WorkflowStore) SetTriggerEnabled(ctx context.Context, workflowName string, triggerID int64, enabled bool) (*gen.WorkflowTrigger, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	wf, err := s.client.Workflow.Query().Where(workflow.NameEQ(workflowName)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, fmt.Errorf("%w: workflow %s", types.ErrNotFound, workflowName)
		}
		return nil, err
	}
	n, err := s.client.WorkflowTrigger.Update().
		Where(
			workflowtrigger.IDEQ(triggerID),
			workflowtrigger.WorkflowIDEQ(wf.ID),
		).
		SetEnabled(enabled).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("%w: trigger %d on workflow %s", types.ErrNotFound, triggerID, workflowName)
	}
	return s.client.WorkflowTrigger.Query().Where(workflowtrigger.IDEQ(triggerID)).Only(ctx)
}

// ListRunsByName returns workflow runs matching the given workflow name.
func (s *WorkflowStore) ListRunsByName(ctx context.Context, name string) ([]*gen.WorkflowRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.WorkflowRun.Query().
		Where(workflowrun.WorkflowNameEQ(name)).
		Order(gen.Desc(workflowrun.FieldCreatedAt)).
		Limit(100).
		All(ctx)
}

// RunLatencyStatsByNames returns success rate and P50/P95 duration stats per workflow name.
// Only completed runs with started_at >= since (when since is non-zero) are included.
// Names without qualifying runs are omitted from the result.
func (s *WorkflowStore) RunLatencyStatsByNames(ctx context.Context, names []string, since time.Time) (map[string]types.RunLatencyStats, error) {
	result := make(map[string]types.RunLatencyStats)
	if s == nil || s.client == nil || len(names) == 0 {
		return result, nil
	}
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		if name != "" {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return result, nil
	}
	q := s.client.WorkflowRun.Query().
		Where(
			workflowrun.WorkflowNameIn(filtered...),
			workflowrun.CompletedAtNotNil(),
		)
	if !since.IsZero() {
		q = q.Where(workflowrun.StartedAtGTE(since))
	}
	runs, err := q.Select(
		workflowrun.FieldWorkflowName,
		workflowrun.FieldStatus,
		workflowrun.FieldStartedAt,
		workflowrun.FieldCompletedAt,
	).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("run latency stats by workflow names: %w", err)
	}
	byName := make(map[string][]runLatencyOutcome)
	for _, run := range runs {
		if run == nil || run.CompletedAt == nil || run.StartedAt.IsZero() {
			continue
		}
		dur := max(run.CompletedAt.Sub(run.StartedAt).Milliseconds(), 0)
		byName[run.WorkflowName] = append(byName[run.WorkflowName], runLatencyOutcome{
			durationMs: dur,
			success:    run.Status == int(schema.WorkflowRunDone),
		})
	}
	for name, outcomes := range byName {
		result[name] = computeRunLatencyStats(outcomes)
	}
	return result, nil
}

// LatestRunStartedAtByNames returns the latest started_at for each workflow name.
// Names without runs are omitted from the result.
func (s *WorkflowStore) LatestRunStartedAtByNames(ctx context.Context, names []string) (map[string]time.Time, error) {
	result := make(map[string]time.Time)
	if s == nil || s.client == nil || len(names) == 0 {
		return result, nil
	}
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		if name != "" {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return result, nil
	}
	runs, err := s.client.WorkflowRun.Query().
		Where(workflowrun.WorkflowNameIn(filtered...)).
		Select(workflowrun.FieldWorkflowName, workflowrun.FieldStartedAt).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("latest workflow run started_at by names: %w", err)
	}
	for _, run := range runs {
		if run == nil {
			continue
		}
		if prev, ok := result[run.WorkflowName]; !ok || run.StartedAt.After(prev) {
			result[run.WorkflowName] = run.StartedAt
		}
	}
	return result, nil
}

// WorkflowStats returns aggregated workflow run statistics for chart rendering.
// name empty = all workflows. since zero = no time filter. groupBy = "day"|"week"|"month".
func (s *WorkflowStore) WorkflowStats(ctx context.Context, name string, since time.Time, groupBy string) (*types.WorkflowStats, error) {
	if s == nil || s.client == nil {
		return emptyWorkflowStats(), nil
	}
	stats := &types.WorkflowStats{}

	var err error
	stats.Summary, err = s.loadWorkflowStatsSummary(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("summary: %w", err)
	}
	stats.SuccessRateTrend, err = s.loadWorkflowSuccessRate(ctx, name, since, groupBy)
	if err != nil {
		return nil, fmt.Errorf("success rate: %w", err)
	}
	stats.DurationDistribution.Workflow, err = s.loadWorkflowDurationBuckets(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("workflow duration: %w", err)
	}
	stats.TriggerSourcePie, err = s.loadWorkflowTriggerSources(ctx, name, since)
	if err != nil {
		return nil, fmt.Errorf("trigger sources: %w", err)
	}
	return stats, nil
}

func (s *WorkflowStore) loadWorkflowStatsSummary(ctx context.Context, name string, since time.Time) (types.WorkflowStatsSummary, error) {
	summary := types.WorkflowStatsSummary{}
	if name == "" {
		count, err := s.client.Workflow.Query().Count(ctx)
		if err != nil {
			return summary, err
		}
		summary.TotalWorkflows = int64(count)
	}

	successful, err := s.countCompletedWorkflowRunsByStatus(ctx, name, since, int(schema.WorkflowRunDone))
	if err != nil {
		return summary, err
	}
	failed, err := s.countCompletedWorkflowRunsByStatus(ctx, name, since, int(schema.WorkflowRunFailed))
	if err != nil {
		return summary, err
	}
	summary.SuccessfulRuns = successful
	summary.FailedRuns = failed
	return summary, nil
}

func (s *WorkflowStore) countCompletedWorkflowRunsByStatus(ctx context.Context, name string, since time.Time, status int) (int64, error) {
	q := s.client.WorkflowRun.Query().
		Where(
			workflowrun.CompletedAtNotNil(),
			workflowrun.StatusEQ(status),
		)
	if name != "" {
		q = q.Where(workflowrun.WorkflowNameEQ(name))
	}
	if !since.IsZero() {
		q = q.Where(workflowrun.StartedAtGTE(since))
	}
	count, err := q.Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (s *WorkflowStore) loadWorkflowSuccessRate(ctx context.Context, name string, since time.Time, groupBy string) ([]types.SuccessRatePoint, error) {
	runs, err := s.fetchCompletedWorkflowRuns(ctx, name, since)
	if err != nil {
		return nil, err
	}
	return computeWorkflowSuccessRate(runs, groupBy), nil
}

func (s *WorkflowStore) fetchCompletedWorkflowRuns(ctx context.Context, name string, since time.Time) ([]*gen.WorkflowRun, error) {
	q := s.client.WorkflowRun.Query().Where(workflowrun.CompletedAtNotNil())
	if name != "" {
		q = q.Where(workflowrun.WorkflowNameEQ(name))
	}
	if !since.IsZero() {
		q = q.Where(workflowrun.StartedAtGTE(since))
	}
	return q.All(ctx)
}

func computeWorkflowSuccessRate(runs []*gen.WorkflowRun, groupBy string) []types.SuccessRatePoint {
	type dayStats struct {
		total   int64
		success int64
	}
	buckets := make(map[string]*dayStats)
	for _, r := range runs {
		if r.CompletedAt == nil {
			continue
		}
		key := dateGroupKey(*r.CompletedAt, groupBy)
		if buckets[key] == nil {
			buckets[key] = &dayStats{}
		}
		buckets[key].total++
		if r.Status == int(schema.WorkflowRunDone) {
			buckets[key].success++
		}
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
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

func (s *WorkflowStore) loadWorkflowDurationBuckets(ctx context.Context, name string, since time.Time) ([]types.DurationEntry, error) {
	runs, err := s.fetchCompletedWorkflowRuns(ctx, name, since)
	if err != nil {
		return nil, err
	}
	result := emptyDurationBuckets()
	for _, r := range runs {
		if r.StartedAt.IsZero() || r.CompletedAt == nil {
			continue
		}
		incrementDurationBucket(result, r.CompletedAt.Sub(r.StartedAt))
	}
	return result, nil
}

func (s *WorkflowStore) loadWorkflowTriggerSources(ctx context.Context, name string, since time.Time) ([]types.TriggerSourceCount, error) {
	q := s.client.WorkflowRun.Query()
	if name != "" {
		q = q.Where(workflowrun.WorkflowNameEQ(name))
	}
	if !since.IsZero() {
		q = q.Where(workflowrun.StartedAtGTE(since))
	}

	type row struct {
		Source string `sql:"trigger_type"`
		Count  int64  `sql:"count"`
	}
	var rows []row

	err := q.GroupBy(workflowrun.FieldTriggerType).
		Aggregate(gen.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	result := map[string]int64{"cron": 0, "webhook": 0, "manual": 0}
	for _, r := range rows {
		result[r.Source] = r.Count
	}
	return []types.TriggerSourceCount{
		{Source: "cron", Count: result["cron"]},
		{Source: "webhook", Count: result["webhook"]},
		{Source: "manual", Count: result["manual"]},
	}, nil
}

func emptyWorkflowStats() *types.WorkflowStats {
	return &types.WorkflowStats{
		Summary: types.WorkflowStatsSummary{},
		TriggerSourcePie: []types.TriggerSourceCount{
			{Source: "cron"}, {Source: "webhook"}, {Source: "manual"},
		},
		DurationDistribution: types.WorkflowDurationDistribution{
			Workflow: emptyDurationBuckets(),
		},
	}
}
