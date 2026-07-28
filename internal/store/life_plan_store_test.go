package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactiondependency"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeactionspec"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeplannode"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
)

func TestLifeStore_CreatePlanNodeAndConfirmHabit(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "user-1", "Ada", "Architect")
	require.NoError(t, err)

	goal, goalSpec, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		NodeType:  "goal",
		Title:     "Ship a product",
		Status:    "Active",
	})
	require.NoError(t, err)
	assert.Nil(t, goalSpec)

	action, spec, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &goal.ID,
		NodeType:  "action",
		Title:     "Write onboarding copy",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:              "habit_candidate",
			TrackingMode:          "consistency",
			IsRepeatable:          true,
			RepeatTrigger:         "time",
			SuggestedCadence:      "daily",
			NeedsUserConfirmation: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "habit_candidate", spec.TaskType)
	assert.True(t, spec.NeedsUserConfirmation)

	spec, err = ls.ConfirmHabitAction(ctx, action.ID)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, "habit", spec.TaskType)
	assert.False(t, spec.NeedsUserConfirmation)
	assert.NotNil(t, spec.ConfirmedAt)

	allNodes, err := ls.ListPlanNodes(ctx, profile.ID)
	require.NoError(t, err)
	assert.Len(t, allNodes, 2)

	allSpecs, err := ls.ListActionSpecs(ctx, profile.ID)
	require.NoError(t, err)
	require.Len(t, allSpecs, 1)
	assert.Equal(t, action.ID, allSpecs[0].PlanNodeID)
}

func TestLifeStore_DeletePlanNodeRemovesDescendants(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "user-2", "Lin", "Architect")
	require.NoError(t, err)

	goal, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		NodeType:  "goal",
		Title:     "Write a book",
		Status:    "Active",
	})
	require.NoError(t, err)

	project, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &goal.ID,
		NodeType:  "project",
		Title:     "Draft chapter one",
		Status:    "Active",
	})
	require.NoError(t, err)

	_, _, err = ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &project.ID,
		NodeType:  "action",
		Title:     "Outline intro",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:      "todo",
			TrackingMode:  "completion",
			RepeatTrigger: "none",
		},
	})
	require.NoError(t, err)

	err = ls.DeletePlanNode(ctx, profile.ID, goal.ID)
	require.NoError(t, err)

	nodes, err := ls.ListPlanNodes(ctx, profile.ID)
	require.NoError(t, err)
	assert.Empty(t, nodes)

	count, err := client.LifeActionSpec.Query().Where(lifeactionspec.TaskTypeEQ("todo")).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestLifeStore_EnsureRecurringOccurrencesAndComplete(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "user-3", "Kai", "Architect")
	require.NoError(t, err)
	goal, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		NodeType:  "goal",
		Title:     "Stay healthy",
		Status:    "Active",
	})
	require.NoError(t, err)
	action, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &goal.ID,
		NodeType:  "action",
		Title:     "Walk daily",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:         "recurring",
			TrackingMode:     "completion",
			IsRepeatable:     true,
			RepeatTrigger:    "time",
			SuggestedCadence: "daily",
		},
	})
	require.NoError(t, err)

	now := time.Date(2026, 7, 28, 15, 0, 0, 0, time.UTC)
	require.NoError(t, ls.EnsureRecurringOccurrences(ctx, profile.ID, now))
	require.NoError(t, ls.EnsureRecurringOccurrences(ctx, profile.ID, now))

	occurrences, err := ls.ListActionOccurrences(ctx, profile.ID, "pending")
	require.NoError(t, err)
	require.Len(t, occurrences, 1)
	assert.Equal(t, action.ID, occurrences[0].PlanNodeID)
	assert.Equal(t, "recurring", occurrences[0].Kind)

	err = ls.CompleteActionOccurrence(ctx, LifeCompleteOccurrenceInput{
		OccurrenceID: occurrences[0].ID,
		ProfileID:    profile.ID,
		PlanNodeID:   action.ID,
		Summary:      action.Title,
	})
	require.NoError(t, err)

	require.NoError(t, ls.EnsureRecurringOccurrences(ctx, profile.ID, now))
	again, err := ls.ListActionOccurrences(ctx, profile.ID, "pending")
	require.NoError(t, err)
	assert.Empty(t, again)

	logs, err := ls.ListActionLogs(ctx, profile.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "occurrence", logs[0].SourceType)
	assert.Equal(t, "Walk daily", logs[0].Summary)

	err = ls.CompleteActionOccurrence(ctx, LifeCompleteOccurrenceInput{
		OccurrenceID: occurrences[0].ID,
		ProfileID:    profile.ID,
		PlanNodeID:   action.ID,
		Summary:      action.Title,
	})
	require.Error(t, err)
}

func TestLifeStore_CreateCheckpointAndAutoComplete(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "user-5", "Nia", "Architect")
	require.NoError(t, err)
	project, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		NodeType:  "goal",
		Title:     "Ship release",
		Status:    "Active",
	})
	require.NoError(t, err)

	todoA, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &project.ID,
		NodeType:  "action",
		Title:     "Write changelog",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:      "todo",
			TrackingMode:  "completion",
			RepeatTrigger: "none",
		},
	})
	require.NoError(t, err)
	todoB, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &project.ID,
		NodeType:  "action",
		Title:     "Tag release",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:      "todo",
			TrackingMode:  "completion",
			RepeatTrigger: "none",
		},
	})
	require.NoError(t, err)
	checkpoint, checkpointSpec, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &project.ID,
		NodeType:  "action",
		Title:     "Release ready",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:      "checkpoint",
			TrackingMode:  "completion",
			RepeatTrigger: "condition",
		},
		DependencyPlanNodeIDs: []int64{todoA.ID, todoB.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, checkpointSpec)
	assert.Equal(t, "checkpoint", checkpointSpec.TaskType)
	_, err = ls.EnsureTodoOccurrence(ctx, profile.ID, todoA.ID)
	require.NoError(t, err)
	_, err = ls.EnsureTodoOccurrence(ctx, profile.ID, todoB.ID)
	require.NoError(t, err)

	deps, err := client.LifeActionDependency.Query().
		Where(lifeactiondependency.ActionPlanNodeIDEQ(checkpoint.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, deps, 2)

	occurrences, err := ls.ListActionOccurrences(ctx, profile.ID, "pending")
	require.NoError(t, err)
	require.Len(t, occurrences, 2)

	err = ls.CompleteActionOccurrence(ctx, LifeCompleteOccurrenceInput{
		OccurrenceID: occurrences[0].ID,
		ProfileID:    profile.ID,
		PlanNodeID:   occurrences[0].PlanNodeID,
		Summary:      "first todo",
	})
	require.NoError(t, err)

	checkpointNode, err := client.LifePlanNode.Get(ctx, checkpoint.ID)
	require.NoError(t, err)
	assert.Equal(t, "Active", checkpointNode.Status)

	err = ls.CompleteActionOccurrence(ctx, LifeCompleteOccurrenceInput{
		OccurrenceID: occurrences[1].ID,
		ProfileID:    profile.ID,
		PlanNodeID:   occurrences[1].PlanNodeID,
		Summary:      "second todo",
	})
	require.NoError(t, err)

	checkpointNode, err = client.LifePlanNode.Get(ctx, checkpoint.ID)
	require.NoError(t, err)
	assert.Equal(t, "Completed", checkpointNode.Status)

	logs, err := ls.ListActionLogs(ctx, profile.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 3)
	assert.Equal(t, "checkpoint", logs[0].SourceType)
	assert.Equal(t, "Release ready", logs[0].Summary)
}

func TestLifeStore_CreateCheckpointRejectsInvalidDependency(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "user-6", "Rin", "Architect")
	require.NoError(t, err)
	goal, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		NodeType:  "goal",
		Title:     "Get fit",
		Status:    "Active",
	})
	require.NoError(t, err)
	projectA, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &goal.ID,
		NodeType:  "project",
		Title:     "Morning routine",
		Status:    "Active",
	})
	require.NoError(t, err)
	projectB, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &goal.ID,
		NodeType:  "project",
		Title:     "Evening routine",
		Status:    "Active",
	})
	require.NoError(t, err)
	todoA, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &projectA.ID,
		NodeType:  "action",
		Title:     "Stretch",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:      "todo",
			TrackingMode:  "completion",
			RepeatTrigger: "none",
		},
	})
	require.NoError(t, err)
	todoB, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &projectB.ID,
		NodeType:  "action",
		Title:     "Read",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:      "todo",
			TrackingMode:  "completion",
			RepeatTrigger: "none",
		},
	})
	require.NoError(t, err)

	_, _, err = ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &projectA.ID,
		NodeType:  "action",
		Title:     "Routine done",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:      "checkpoint",
			TrackingMode:  "completion",
			RepeatTrigger: "condition",
		},
		DependencyPlanNodeIDs: []int64{todoA.ID, todoB.ID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency action must share parent")

	count, err := client.LifePlanNode.Query().
		Where(lifeplannode.LifeProfileIDEQ(profile.ID), lifeplannode.TitleEQ("Routine done")).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestLifeStore_UpsertHabitCheckinIsIdempotent(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "user-4", "Mia", "Architect")
	require.NoError(t, err)
	goal, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		NodeType:  "goal",
		Title:     "Keep writing",
		Status:    "Active",
	})
	require.NoError(t, err)
	action, _, err := ls.CreatePlanNode(ctx, LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &goal.ID,
		NodeType:  "action",
		Title:     "Write every morning",
		Status:    "Active",
		ActionSpec: &LifePlanActionSpecInput{
			TaskType:         "habit",
			TrackingMode:     "consistency",
			IsRepeatable:     true,
			RepeatTrigger:    "time",
			SuggestedCadence: "daily",
		},
	})
	require.NoError(t, err)

	checkinAt := time.Date(2026, 7, 28, 9, 15, 0, 0, time.UTC)
	first, err := ls.UpsertHabitCheckin(ctx, LifeHabitCheckinInput{
		ProfileID:  profile.ID,
		PlanNodeID: action.ID,
		CheckinAt:  checkinAt,
		Status:     "done",
		Summary:    action.Title,
	})
	require.NoError(t, err)
	second, err := ls.UpsertHabitCheckin(ctx, LifeHabitCheckinInput{
		ProfileID:  profile.ID,
		PlanNodeID: action.ID,
		CheckinAt:  checkinAt.Add(2 * time.Hour),
		Status:     "done",
		Summary:    action.Title,
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	checkins, err := ls.ListHabitCheckins(ctx, profile.ID, action.ID, time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 28, 23, 59, 59, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, checkins, 1)

	logs, err := ls.ListActionLogs(ctx, profile.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "habit_checkin", logs[0].SourceType)
}
