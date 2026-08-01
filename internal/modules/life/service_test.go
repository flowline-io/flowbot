package life

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestParseLoreInventoryID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		payload    map[string]any
		wantID     int64
		wantPoison bool
	}{
		{
			name:       "valid",
			payload:    map[string]any{"inventory_id": int64(42), "type": "life.inventory.lore_requested"},
			wantID:     42,
			wantPoison: false,
		},
		{
			name:       "float id from json",
			payload:    map[string]any{"inventory_id": float64(7)},
			wantID:     7,
			wantPoison: false,
		},
		{
			name:       "missing id",
			payload:    map[string]any{"type": "life.inventory.lore_requested"},
			wantID:     0,
			wantPoison: true,
		},
		{
			name:       "zero id",
			payload:    map[string]any{"inventory_id": int64(0)},
			wantID:     0,
			wantPoison: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			row := &gen.EventOutbox{EventID: "e1", Payload: tt.payload}
			id, poison := parseLoreInventoryID(row)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.wantPoison, poison)
		})
	}
}

func TestResolveDropEquip_NeedLore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		questType string
		diff      string
		wantLore  bool
	}{
		{name: "daily common", questType: "Daily", diff: "B", wantLore: false},
		{name: "boss", questType: "Boss", diff: "A", wantLore: true},
		{name: "ss", questType: "One-Time", diff: "SS", wantLore: true},
		{name: "sss", questType: "One-Time", diff: "SSS", wantLore: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			need := pkglife.NeedsInstanceLore(tt.questType, tt.diff)
			assert.Equal(t, tt.wantLore, need)
		})
	}
}

func TestClassifyActionInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		in        *ActionInput
		wantType  string
		wantMode  string
		wantNeeds bool
	}{
		{
			name:      "todo by default",
			in:        &ActionInput{},
			wantType:  "todo",
			wantMode:  "completion",
			wantNeeds: false,
		},
		{
			name: "recurring when repeatable completion",
			in: &ActionInput{
				IsRepeatable: true,
				TrackingMode: "completion",
			},
			wantType:  "recurring",
			wantMode:  "completion",
			wantNeeds: false,
		},
		{
			name: "habit candidate when consistency",
			in: &ActionInput{
				IsRepeatable: true,
				TrackingMode: "consistency",
			},
			wantType:  "habit_candidate",
			wantMode:  "consistency",
			wantNeeds: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyActionInput(tt.in)
			assert.Equal(t, tt.wantType, got.TaskType)
			assert.Equal(t, tt.wantMode, got.TrackingMode)
			assert.Equal(t, tt.wantNeeds, got.NeedsUserConfirmation)
			assert.Equal(t, 25, got.BaseExpReward)
			assert.Equal(t, 8, got.BaseGoldReward)
			assert.Equal(t, "C", got.Difficulty)
		})
	}
}

func TestBuildPlanTree(t *testing.T) {
	t.Parallel()
	parentID := int64(1)
	projectID := int64(2)
	nodes := []*gen.LifePlanNode{
		{ID: 1, Flag: "goal-1", NodeType: "goal", Title: "Ship app"},
		{ID: 2, Flag: "proj-1", ParentID: &parentID, NodeType: "project", Title: "Backend"},
		{ID: 3, Flag: "act-1", ParentID: &projectID, NodeType: "action", Title: "Write handlers"},
	}
	specs := []*gen.LifeActionSpec{
		{PlanNodeID: 3, TaskType: "todo", TrackingMode: "completion"},
	}
	tree := buildPlanTree(nodes, specs)
	if assert.Len(t, tree, 1) {
		assert.Equal(t, "goal-1", tree[0].Node.Flag)
		if assert.Len(t, tree[0].Children, 1) {
			assert.Equal(t, "proj-1", tree[0].Children[0].Node.Flag)
			if assert.Len(t, tree[0].Children[0].Children, 1) {
				action := tree[0].Children[0].Children[0]
				assert.Equal(t, "act-1", action.Node.Flag)
				assert.NotNil(t, action.Action)
				assert.Equal(t, "todo", action.Action.TaskType)
			}
		}
	}
}

func TestIsAllowedPlanChild(t *testing.T) {
	t.Parallel()
	assert.True(t, isAllowedPlanChild("goal", "milestone"))
	assert.True(t, isAllowedPlanChild("goal", "project"))
	assert.True(t, isAllowedPlanChild("project", "action"))
	assert.False(t, isAllowedPlanChild("goal", "action"))
	assert.False(t, isAllowedPlanChild("project", "project"))
	assert.False(t, isAllowedPlanChild("action", "goal"))
}

func TestImportGoalBreakdownIsAtomic(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	suggestion := (*lifecap.GoalBreakdownSuggestion)(nil)

	err := svc.ImportGoalBreakdown(ctx, "review-user", suggestion)
	require.Error(t, err)

	profile, err := svc.EnsureProfile(ctx, "review-user", "", "")
	require.NoError(t, err)
	nodes, err := svc.store.ListPlanNodes(ctx, profile.ID)
	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestImportGoalBreakdownNormalizesInvalidHierarchy(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	suggestion := &lifecap.GoalBreakdownSuggestion{
		NodeType: "goal",
		Title:    "Ship V2",
		Children: []lifecap.GoalBreakdownSuggestion{
			{
				NodeType: "action",
				Title:    "Prototype dialogue",
				Children: []lifecap.GoalBreakdownSuggestion{
					{
						NodeType: "action",
						Title:    "Write dialogue core",
						Action: &lifecap.GoalBreakdownActionSuggestion{
							IsRepeatable:  false,
							TrackingMode:  "completion",
							RepeatTrigger: "none",
						},
					},
				},
			},
		},
	}

	err := svc.ImportGoalBreakdown(ctx, "normalize-user", suggestion)
	require.NoError(t, err)

	profile, err := svc.EnsureProfile(ctx, "normalize-user", "", "")
	require.NoError(t, err)
	nodes, err := svc.store.ListPlanNodes(ctx, profile.ID)
	require.NoError(t, err)
	require.Len(t, nodes, 3)
	assert.Equal(t, "goal", nodes[0].NodeType)
	assert.Equal(t, "project", nodes[1].NodeType)
	assert.Equal(t, "action", nodes[2].NodeType)
	specs, err := svc.store.ListActionSpecs(ctx, profile.ID)
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, 25, specs[0].BaseExpReward)
	assert.Equal(t, 8, specs[0].BaseGoldReward)
	assert.Equal(t, "C", specs[0].Difficulty)
}

func TestSubmitQuestEvidenceStoresPendingQuestProof(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "evidence-user", "", "")
	require.NoError(t, err)
	chars, err := svc.store.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := svc.store.CreateSkill(ctx, profile.ID, chars[0].ID, "Systems Design", 0.5)
	require.NoError(t, err)
	quest, err := svc.store.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "Ship quest adjudication",
		Prompt:                "Ship the evidence flow",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "A",
		BaseExpReward:         150,
		BaseGoldReward:        40,
		DropTier:              "Epic",
	})
	require.NoError(t, err)

	view, err := svc.SubmitQuestEvidence(ctx, "evidence-user", quest.Flag, "note", "Finished the flow and added tests.", "")
	require.NoError(t, err)
	assert.Equal(t, "note", view.SourceType)
	assert.Contains(t, view.Content, "added tests")

	rows, err := svc.store.ListEvidenceByQuest(ctx, profile.ID, quest.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "note", rows[0].SourceType)
}

func TestDismissQuestMarksPendingWithoutRust(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "dismiss-user", "", "")
	require.NoError(t, err)
	chars, err := svc.store.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := svc.store.CreateSkill(ctx, profile.ID, chars[0].ID, "Focus", 0.5)
	require.NoError(t, err)
	quest, err := svc.store.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "Stuck quest to dismiss",
		Prompt:                "Clear a stuck pending quest",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "E",
		BaseExpReward:         10,
		BaseGoldReward:        3,
		DropTier:              "Common",
	})
	require.NoError(t, err)

	err = svc.DismissQuest(ctx, "dismiss-user", quest.Flag)
	require.NoError(t, err)

	got, err := svc.store.GetQuestByFlag(ctx, profile.ID, quest.Flag)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Dismissed", got.Status)

	slots, err := svc.store.GetEquippedSlots(ctx, profile.ID)
	require.NoError(t, err)
	assert.Nil(t, slots.TarnishedUntil)

	err = svc.DismissQuest(ctx, "dismiss-user", quest.Flag)
	require.Error(t, err)
}

func TestSummarizeEvidenceKeepsValidUTF8(t *testing.T) {
	t.Parallel()
	long := "Notice of Assessment from IRAS: " + strings.Repeat("x", 90) + "完成报税流程验证并核对税额"
	sum := summarizeEvidence(long)
	require.True(t, utf8.ValidString(sum))
	assert.LessOrEqual(t, len([]rune(sum)), 120)
	assert.Contains(t, sum, "Notice of Assessment")
}

func TestSubmitQuestEvidenceAcceptsLongMixedScriptContent(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "mixed-evidence-user", "", "")
	require.NoError(t, err)
	chars, err := svc.store.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := svc.store.CreateSkill(ctx, profile.ID, chars[0].ID, "Tax", 0.5)
	require.NoError(t, err)
	quest, err := svc.store.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "完成上笔报税流程",
		Prompt:                "完成上个的报税流程",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "B",
		BaseExpReward:         40,
		BaseGoldReward:        12,
		DropTier:              "Rare",
	})
	require.NoError(t, err)

	content := "Notice of Assessment from IRAS: " + strings.Repeat("x", 90) + "完成报税流程验证并核对税额"
	view, err := svc.SubmitQuestEvidence(ctx, "mixed-evidence-user", quest.Flag, "note", content, "")
	require.NoError(t, err)
	require.True(t, utf8.ValidString(view.Summary))
	assert.Equal(t, content, view.Content)
}

func TestApplyQuestAdjudicationCompletesQuest(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "apply-user", "", "")
	require.NoError(t, err)
	chars, err := svc.store.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := svc.store.CreateSkill(ctx, profile.ID, chars[0].ID, "Execution", 0.5)
	require.NoError(t, err)
	quest, err := svc.store.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "Ship DM MVP",
		Prompt:                "Ship the first DM MVP",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "B",
		BaseExpReward:         80,
		BaseGoldReward:        25,
		DropTier:              "Rare",
	})
	require.NoError(t, err)
	adjudication, err := svc.store.CreateAdjudication(ctx, store.LifeAdjudicationInput{
		ProfileID:          profile.ID,
		QuestID:            quest.ID,
		Status:             "suggested",
		Verdict:            "completed",
		Reason:             "The evidence was accepted.",
		SuggestedExp:       quest.BaseExpReward,
		SuggestedGold:      quest.BaseGoldReward,
		SuggestedNextSteps: []string{"Write a retro"},
		EvidenceSnapshot:   []map[string]any{{"summary": "Finished implementation"}},
	})
	require.NoError(t, err)

	err = svc.ApplyQuestAdjudication(ctx, "apply-user", quest.Flag, adjudication.Flag)
	require.NoError(t, err)

	gotQuest, err := svc.store.GetQuestByFlag(ctx, profile.ID, quest.Flag)
	require.NoError(t, err)
	require.NotNil(t, gotQuest)
	assert.Equal(t, "Completed", gotQuest.Status)

	gotAdj, err := svc.store.GetAdjudicationByFlag(ctx, profile.ID, adjudication.Flag)
	require.NoError(t, err)
	require.NotNil(t, gotAdj)
	assert.Equal(t, "applied", gotAdj.Status)
}

func TestCompleteActionOccurrenceGrantsRewards(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "action-reward-user", "", "")
	require.NoError(t, err)
	goal, _, err := svc.store.CreatePlanNode(ctx, store.LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		NodeType:  "goal",
		Title:     "Ship plan rewards",
		Status:    "Active",
	})
	require.NoError(t, err)
	action, spec, err := svc.store.CreatePlanNode(ctx, store.LifeCreatePlanNodeInput{
		ProfileID: profile.ID,
		ParentID:  &goal.ID,
		NodeType:  "action",
		Title:     "Wire rewards",
		Status:    "Active",
		ActionSpec: &store.LifePlanActionSpecInput{
			TaskType:       "todo",
			TrackingMode:   "completion",
			RepeatTrigger:  "none",
			Difficulty:     "B",
			BaseExpReward:  80,
			BaseGoldReward: 25,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, spec)
	occurrence, err := svc.store.EnsureTodoOccurrence(ctx, profile.ID, action.ID)
	require.NoError(t, err)

	err = svc.CompleteActionOccurrence(ctx, "action-reward-user", occurrence.Flag)
	require.NoError(t, err)

	updated, err := svc.store.GetProfileByUserID(ctx, "action-reward-user")
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 25, updated.Gold)
	assert.Equal(t, int64(80), updated.Exp)

	logs, err := svc.store.ListActionLogs(ctx, profile.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, 80, logs[0].GainedExp)
	assert.Equal(t, 25, logs[0].GainedGold)
}

func TestListCompletedQuestsPageAndActionLogsPage(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	svc := NewService(store.NewLifeStore(client))
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "quests-logs-page-user", "", "")
	require.NoError(t, err)
	chars, err := svc.store.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := svc.store.CreateSkill(ctx, profile.ID, chars[0].ID, "Focus", 0.5)
	require.NoError(t, err)

	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i := range 5 {
		quest, err := svc.store.CreateQuest(ctx, &gen.LifeQuest{
			LifeProfileID:         profile.ID,
			SkillID:               skill.ID,
			Title:                 fmt.Sprintf("Done %d", i),
			Prompt:                "prompt",
			Type:                  "One-Time",
			AiEvaluatedDifficulty: "B",
			BaseExpReward:         10,
			BaseGoldReward:        5,
			DropTier:              "Common",
		})
		require.NoError(t, err)
		_, err = client.LifeQuest.UpdateOneID(quest.ID).
			SetStatus("Completed").
			SetCompletedAt(base.Add(time.Duration(i) * time.Minute)).
			Save(ctx)
		require.NoError(t, err)
		_, err = client.LifeActionLog.Create().
			SetFlag(types.Id()).
			SetLifeProfileID(profile.ID).
			SetQuestID(quest.ID).
			SetSourceType("quest").
			SetSummary(fmt.Sprintf("Log %d", i)).
			SetGainedExp(10).
			SetGainedGold(5).
			SetCreatedAt(base.Add(time.Duration(i) * time.Minute)).
			Save(ctx)
		require.NoError(t, err)
	}

	quests, total, err := svc.ListCompletedQuestsPage(ctx, "quests-logs-page-user", 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, quests, 2)
	assert.Equal(t, "Done 2", quests[0].Title)

	clamped, total, err := svc.ListCompletedQuestsPage(ctx, "quests-logs-page-user", 99, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, clamped, 1)
	assert.Equal(t, "Done 0", clamped[0].Title)

	logs, logTotal, err := svc.ListActionLogsPage(ctx, "quests-logs-page-user", 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, logTotal)
	require.Len(t, logs, 2)
	assert.Equal(t, "Done 2", logs[0].QuestTitle)
	assert.Equal(t, "Done 1", logs[1].QuestTitle)
}
