package life

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
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
			q := &gen.LifeQuest{Type: tt.questType, AiEvaluatedDifficulty: tt.diff}
			need := q.Type == "Boss" || q.AiEvaluatedDifficulty == "SSS" || q.AiEvaluatedDifficulty == "SS"
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
