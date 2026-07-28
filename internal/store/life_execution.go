package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactiondependency"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactionlog"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactionoccurrence"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactionspec"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifehabitcheckin"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeplannode"
	"github.com/flowline-io/flowbot/pkg/types"
)

// LifeCreateOccurrenceInput is the write shape for creating one action occurrence.
type LifeCreateOccurrenceInput struct {
	ProfileID          int64
	PlanNodeID         int64
	Kind               string
	State              string
	DueAt              time.Time
	CadenceSnapshot    string
	SourceOccurrenceID *int64
}

// LifeCompleteOccurrenceInput is the transaction write shape for one completed occurrence.
type LifeCompleteOccurrenceInput struct {
	OccurrenceID int64
	ProfileID    int64
	PlanNodeID   int64
	Summary      string
	GainedExp    int
	GainedGold   int
}

// LifeSkipOccurrenceInput is the write shape for skipping one occurrence.
type LifeSkipOccurrenceInput struct {
	OccurrenceID int64
	State        string
}

// LifeHabitCheckinInput is the write shape for a habit check-in.
type LifeHabitCheckinInput struct {
	ProfileID  int64
	PlanNodeID int64
	CheckinAt  time.Time
	Status     string
	Note       string
	Summary    string
}

type checkpointCompletionInput struct {
	ProfileID  int64
	PlanNodeID int64
	Summary    string
}

// CreateActionOccurrence inserts one occurrence row.
func (s *LifeStore) CreateActionOccurrence(ctx context.Context, in LifeCreateOccurrenceInput) (*gen.LifeActionOccurrence, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	kind := normalizeOccurrenceKind(in.Kind)
	state := normalizeOccurrenceState(in.State)
	builder := s.client.LifeActionOccurrence.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetPlanNodeID(in.PlanNodeID).
		SetKind(kind).
		SetState(state).
		SetDueAt(in.DueAt).
		SetCadenceSnapshot(strings.TrimSpace(in.CadenceSnapshot))
	if in.SourceOccurrenceID != nil {
		builder = builder.SetSourceOccurrenceID(*in.SourceOccurrenceID)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create occurrence: %w", err)
	}
	return row, nil
}

// EnsureRecurringOccurrences lazily creates due daily/weekly occurrences.
func (s *LifeStore) EnsureRecurringOccurrences(ctx context.Context, profileID int64, now time.Time) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	nodes, err := s.ListPlanNodes(ctx, profileID)
	if err != nil {
		return err
	}
	specs, err := s.ListActionSpecs(ctx, profileID)
	if err != nil {
		return err
	}
	nodeByID := make(map[int64]*gen.LifePlanNode, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	for _, spec := range specs {
		if spec.TaskType != "recurring" {
			continue
		}
		node := nodeByID[spec.PlanNodeID]
		if node == nil || node.Status != "Active" {
			continue
		}
		dueAt, ok := occurrenceDueAt(spec.SuggestedCadence, now)
		if !ok {
			continue
		}
		exists, err := s.client.LifeActionOccurrence.Query().
			Where(
				lifeactionoccurrence.LifeProfileIDEQ(profileID),
				lifeactionoccurrence.PlanNodeIDEQ(node.ID),
				lifeactionoccurrence.KindEQ("recurring"),
				lifeactionoccurrence.DueAtEQ(dueAt),
			).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("life: check occurrence: %w", err)
		}
		if exists {
			continue
		}
		if _, err := s.CreateActionOccurrence(ctx, LifeCreateOccurrenceInput{
			ProfileID:       profileID,
			PlanNodeID:      node.ID,
			Kind:            "recurring",
			State:           "pending",
			DueAt:           dueAt,
			CadenceSnapshot: spec.SuggestedCadence,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ListActionOccurrences returns occurrences for a profile, optional state filter.
func (s *LifeStore) ListActionOccurrences(ctx context.Context, profileID int64, state string) ([]*gen.LifeActionOccurrence, error) {
	if !s.ready() {
		return nil, nil
	}
	q := s.client.LifeActionOccurrence.Query().
		Where(lifeactionoccurrence.LifeProfileIDEQ(profileID)).
		Order(gen.Asc(lifeactionoccurrence.FieldDueAt))
	if state != "" {
		q = q.Where(lifeactionoccurrence.StateEQ(state))
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list occurrences: %w", err)
	}
	return rows, nil
}

// GetActionOccurrenceByFlag returns one occurrence by flag scoped to profile.
func (s *LifeStore) GetActionOccurrenceByFlag(ctx context.Context, profileID int64, flag string) (*gen.LifeActionOccurrence, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifeActionOccurrence.Query().
		Where(lifeactionoccurrence.LifeProfileIDEQ(profileID), lifeactionoccurrence.FlagEQ(flag)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("life: get occurrence: %w", err)
	}
	return row, nil
}

// CompleteActionOccurrence completes one occurrence and writes an action log.
func (s *LifeStore) CompleteActionOccurrence(ctx context.Context, in LifeCompleteOccurrenceInput) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("life: begin occurrence tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	now := time.Now()
	affected, err := tx.LifeActionOccurrence.Update().
		Where(lifeactionoccurrence.IDEQ(in.OccurrenceID), lifeactionoccurrence.StateEQ("pending")).
		SetState("completed").
		SetCompletedAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("life: complete occurrence: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("life: occurrence is not pending")
	}
	if _, err := tx.LifeActionLog.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetPlanNodeID(in.PlanNodeID).
		SetSourceType("occurrence").
		SetSummary(in.Summary).
		SetGainedExp(in.GainedExp).
		SetGainedGold(in.GainedGold).
		Save(ctx); err != nil {
		return fmt.Errorf("life: occurrence action log: %w", err)
	}
	if err := completeReadyCheckpoints(ctx, tx.Client(), checkpointCompletionInput{
		ProfileID:  in.ProfileID,
		PlanNodeID: in.PlanNodeID,
		Summary:    in.Summary,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("life: commit occurrence: %w", err)
	}
	committed = true
	return nil
}

// SkipActionOccurrence marks one occurrence skipped or missed.
func (s *LifeStore) SkipActionOccurrence(ctx context.Context, in LifeSkipOccurrenceInput) error {
	if !s.ready() {
		return fmt.Errorf("life: store not available")
	}
	affected, err := s.client.LifeActionOccurrence.Update().
		Where(lifeactionoccurrence.IDEQ(in.OccurrenceID), lifeactionoccurrence.StateEQ("pending")).
		SetState(normalizeOccurrenceState(in.State)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("life: skip occurrence: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("life: occurrence is not pending")
	}
	return nil
}

// ListHabitCheckins returns habit checkins for a node in a time window.
func (s *LifeStore) ListHabitCheckins(ctx context.Context, profileID, planNodeID int64, from, to time.Time) ([]*gen.LifeHabitCheckin, error) {
	if !s.ready() {
		return nil, nil
	}
	q := s.client.LifeHabitCheckin.Query().
		Where(lifehabitcheckin.LifeProfileIDEQ(profileID), lifehabitcheckin.PlanNodeIDEQ(planNodeID)).
		Order(gen.Asc(lifehabitcheckin.FieldCheckinDate))
	if !from.IsZero() {
		q = q.Where(lifehabitcheckin.CheckinDateGTE(from))
	}
	if !to.IsZero() {
		q = q.Where(lifehabitcheckin.CheckinDateLTE(to))
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: list habit checkins: %w", err)
	}
	return rows, nil
}

// UpsertHabitCheckin creates or refreshes a daily habit checkin and writes an action log.
func (s *LifeStore) UpsertHabitCheckin(ctx context.Context, in LifeHabitCheckinInput) (*gen.LifeHabitCheckin, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	day := startOfDayUTC(in.CheckinAt)
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "done"
	}
	existing, err := s.client.LifeHabitCheckin.Query().
		Where(lifehabitcheckin.PlanNodeIDEQ(in.PlanNodeID), lifehabitcheckin.CheckinDateEQ(day)).
		Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !gen.IsNotFound(err) {
		return nil, fmt.Errorf("life: get habit checkin: %w", err)
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: begin habit tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	row, err := tx.LifeHabitCheckin.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetPlanNodeID(in.PlanNodeID).
		SetCheckinDate(day).
		SetStatus(status).
		SetNote(strings.TrimSpace(in.Note)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("life: create habit checkin: %w", err)
	}
	if _, err := tx.LifeActionLog.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetPlanNodeID(in.PlanNodeID).
		SetSourceType("habit_checkin").
		SetSummary(in.Summary).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("life: habit action log: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("life: commit habit tx: %w", err)
	}
	committed = true
	return row, nil
}

// EnsureTodoOccurrence creates a pending one-time occurrence if missing.
func (s *LifeStore) EnsureTodoOccurrence(ctx context.Context, profileID, planNodeID int64) (*gen.LifeActionOccurrence, error) {
	if !s.ready() {
		return nil, fmt.Errorf("life: store not available")
	}
	existing, err := s.client.LifeActionOccurrence.Query().
		Where(
			lifeactionoccurrence.LifeProfileIDEQ(profileID),
			lifeactionoccurrence.PlanNodeIDEQ(planNodeID),
			lifeactionoccurrence.KindEQ("one_time"),
			lifeactionoccurrence.StateEQ("pending"),
		).Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !gen.IsNotFound(err) {
		return nil, fmt.Errorf("life: get todo occurrence: %w", err)
	}
	return s.CreateActionOccurrence(ctx, LifeCreateOccurrenceInput{
		ProfileID:       profileID,
		PlanNodeID:      planNodeID,
		Kind:            "one_time",
		State:           "pending",
		DueAt:           time.Now().UTC(),
		CadenceSnapshot: "once",
	})
}

// GetPlanNode fetches a plan node by id.
func (s *LifeStore) GetPlanNode(ctx context.Context, id int64) (*gen.LifePlanNode, error) {
	if !s.ready() {
		return nil, nil
	}
	row, err := s.client.LifePlanNode.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("life: get plan node by id: %w", err)
	}
	return row, nil
}

// ListRecurringActionSpecs returns active recurring specs and their nodes.
func (s *LifeStore) ListRecurringActionSpecs(ctx context.Context, profileID int64) ([]*gen.LifeActionSpec, map[int64]*gen.LifePlanNode, error) {
	if !s.ready() {
		return nil, nil, nil
	}
	nodes, err := s.client.LifePlanNode.Query().
		Where(lifeplannode.LifeProfileIDEQ(profileID), lifeplannode.NodeTypeEQ("action"), lifeplannode.StatusEQ("Active")).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("life: list recurring nodes: %w", err)
	}
	nodeByID := make(map[int64]*gen.LifePlanNode, len(nodes))
	ids := make([]int64, 0, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
		ids = append(ids, node.ID)
	}
	if len(ids) == 0 {
		return nil, nodeByID, nil
	}
	specs, err := s.client.LifeActionSpec.Query().
		Where(lifeactionspec.PlanNodeIDIn(ids...), lifeactionspec.TaskTypeEQ("recurring")).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("life: list recurring specs: %w", err)
	}
	return specs, nodeByID, nil
}

func (s *LifeStore) createActionDependencies(ctx context.Context, profileID, actionPlanNodeID int64, dependencyPlanNodeIDs []int64) error {
	if len(dependencyPlanNodeIDs) == 0 {
		return nil
	}
	actionNode, err := s.getCheckpointActionNode(ctx, actionPlanNodeID)
	if err != nil {
		return err
	}
	seen := make(map[int64]struct{}, len(dependencyPlanNodeIDs))
	for _, dependencyID := range dependencyPlanNodeIDs {
		if _, ok := seen[dependencyID]; ok {
			continue
		}
		seen[dependencyID] = struct{}{}
		if err := s.createActionDependency(ctx, profileID, actionNode, dependencyID); err != nil {
			return err
		}
	}
	return nil
}

func (s *LifeStore) getCheckpointActionNode(ctx context.Context, actionPlanNodeID int64) (*gen.LifePlanNode, error) {
	actionNode, err := s.client.LifePlanNode.Get(ctx, actionPlanNodeID)
	if err != nil {
		return nil, fmt.Errorf("life: get checkpoint action: %w", err)
	}
	if actionNode.ParentID == nil {
		return nil, fmt.Errorf("life: checkpoint action requires parent")
	}
	actionSpec, err := s.GetActionSpecByPlanNodeID(ctx, actionPlanNodeID)
	if err != nil {
		return nil, err
	}
	if actionSpec == nil || actionSpec.TaskType != "checkpoint" {
		return nil, fmt.Errorf("life: dependencies require checkpoint action")
	}
	return actionNode, nil
}

func (s *LifeStore) createActionDependency(ctx context.Context, profileID int64, actionNode *gen.LifePlanNode, dependencyID int64) error {
	if dependencyID == actionNode.ID {
		return fmt.Errorf("life: checkpoint cannot depend on itself")
	}
	dependencyNode, err := s.client.LifePlanNode.Get(ctx, dependencyID)
	if err != nil {
		return fmt.Errorf("life: get dependency action: %w", err)
	}
	if dependencyNode.LifeProfileID != profileID {
		return fmt.Errorf("life: dependency action not found")
	}
	if dependencyNode.ParentID == nil || *dependencyNode.ParentID != *actionNode.ParentID {
		return fmt.Errorf("life: dependency action must share parent")
	}
	dependencySpec, err := s.GetActionSpecByPlanNodeID(ctx, dependencyID)
	if err != nil {
		return err
	}
	if dependencySpec == nil || dependencySpec.TaskType != "todo" {
		return fmt.Errorf("life: dependency action must be todo")
	}
	if _, err := s.client.LifeActionDependency.Create().
		SetActionPlanNodeID(actionNode.ID).
		SetDependsOnPlanNodeID(dependencyID).
		Save(ctx); err != nil {
		return fmt.Errorf("life: create action dependency: %w", err)
	}
	return nil
}

func completeReadyCheckpoints(ctx context.Context, client *gen.Client, in checkpointCompletionInput) error {
	rows, err := client.LifeActionDependency.Query().
		Where(lifeactiondependency.DependsOnPlanNodeIDEQ(in.PlanNodeID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("life: list action dependencies: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}
	checkpointIDs := make([]int64, 0, len(rows))
	seen := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.ActionPlanNodeID]; ok {
			continue
		}
		seen[row.ActionPlanNodeID] = struct{}{}
		checkpointIDs = append(checkpointIDs, row.ActionPlanNodeID)
	}
	for _, checkpointID := range checkpointIDs {
		ready, summary, err := checkpointReadyForCompletion(ctx, client, in.ProfileID, checkpointID)
		if err != nil {
			return err
		}
		if !ready {
			continue
		}
		if err := completeCheckpointAction(ctx, client, checkpointCompletionInput{
			ProfileID:  in.ProfileID,
			PlanNodeID: checkpointID,
			Summary:    summary,
		}); err != nil {
			return err
		}
	}
	return nil
}

func checkpointReadyForCompletion(ctx context.Context, client *gen.Client, profileID, checkpointID int64) (bool, string, error) {
	node, err := client.LifePlanNode.Get(ctx, checkpointID)
	if err != nil {
		return false, "", fmt.Errorf("life: get checkpoint node: %w", err)
	}
	if node.LifeProfileID != profileID {
		return false, "", fmt.Errorf("life: checkpoint action not found")
	}
	if strings.EqualFold(node.Status, "Completed") {
		return false, "", nil
	}
	spec, err := client.LifeActionSpec.Query().
		Where(lifeactionspec.PlanNodeIDEQ(checkpointID)).
		Only(ctx)
	if err != nil {
		return false, "", fmt.Errorf("life: get checkpoint spec: %w", err)
	}
	if spec.TaskType != "checkpoint" {
		return false, "", nil
	}
	rows, err := client.LifeActionDependency.Query().
		Where(lifeactiondependency.ActionPlanNodeIDEQ(checkpointID)).
		All(ctx)
	if err != nil {
		return false, "", fmt.Errorf("life: list checkpoint dependencies: %w", err)
	}
	if len(rows) == 0 {
		return false, "", fmt.Errorf("life: checkpoint dependencies required")
	}
	for _, row := range rows {
		exists, err := client.LifeActionLog.Query().
			Where(
				lifeactionlog.LifeProfileIDEQ(profileID),
				lifeactionlog.PlanNodeIDEQ(row.DependsOnPlanNodeID),
				lifeactionlog.SourceTypeEQ("occurrence"),
			).
			Exist(ctx)
		if err != nil {
			return false, "", fmt.Errorf("life: check dependency completion: %w", err)
		}
		if !exists {
			return false, "", nil
		}
	}
	return true, node.Title, nil
}

func completeCheckpointAction(ctx context.Context, client *gen.Client, in checkpointCompletionInput) error {
	affected, err := client.LifePlanNode.Update().
		Where(
			lifeplannode.IDEQ(in.PlanNodeID),
			lifeplannode.LifeProfileIDEQ(in.ProfileID),
			lifeplannode.StatusNEQ("Completed"),
		).
		SetStatus("Completed").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("life: complete checkpoint action: %w", err)
	}
	if affected == 0 {
		return nil
	}
	if _, err := client.LifeActionLog.Create().
		SetFlag(types.Id()).
		SetLifeProfileID(in.ProfileID).
		SetPlanNodeID(in.PlanNodeID).
		SetSourceType("checkpoint").
		SetSummary(strings.TrimSpace(in.Summary)).
		Save(ctx); err != nil {
		return fmt.Errorf("life: checkpoint action log: %w", err)
	}
	return completeReadyCheckpoints(ctx, client, in)
}

func normalizeOccurrenceKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "recurring":
		return "recurring"
	default:
		return "one_time"
	}
}

func normalizeOccurrenceState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "completed":
		return "completed"
	case "skipped":
		return "skipped"
	case "missed":
		return "missed"
	case "cancelled":
		return "cancelled"
	default:
		return "pending"
	}
}

func occurrenceDueAt(cadence string, now time.Time) (time.Time, bool) {
	switch strings.ToLower(strings.TrimSpace(cadence)) {
	case "daily":
		return startOfDayUTC(now), true
	case "weekly":
		day := startOfDayUTC(now)
		offset := int(day.Weekday())
		if offset == 0 {
			offset = 7
		}
		return day.AddDate(0, 0, -(offset - 1)), true
	default:
		return time.Time{}, false
	}
}

func startOfDayUTC(t time.Time) time.Time {
	if t.IsZero() {
		t = time.Now()
	}
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
