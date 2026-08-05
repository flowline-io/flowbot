package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
)

func TestLifeStore_CreateEvidenceAndLatestAdjudication(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "quest-dm-user", "Ada", "Architect")
	require.NoError(t, err)
	characteristic, err := ls.CreateCharacteristic(ctx, profile.ID, "INT", "Intelligence")
	require.NoError(t, err)
	skill, err := ls.CreateSkill(ctx, profile.ID, characteristic.ID, "Systems Design", 0.5)
	require.NoError(t, err)
	quest, err := ls.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "Ship AI DM",
		Prompt:                "Ship the first adjudication flow",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "A",
		BaseExpReward:         150,
		BaseGoldReward:        40,
		DropTier:              "Epic",
	})
	require.NoError(t, err)

	evidence, err := ls.CreateEvidence(ctx, LifeEvidenceInput{
		ProfileID:  profile.ID,
		QuestID:    &quest.ID,
		SourceType: "note",
		Content:    "Implemented the flow and wrote tests.",
		Summary:    "Implemented the flow and wrote tests.",
	})
	require.NoError(t, err)
	assert.Equal(t, quest.ID, *evidence.QuestID)

	rows, err := ls.ListEvidenceByQuest(ctx, profile.ID, quest.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, evidence.Flag, rows[0].Flag)

	first, err := ls.CreateAdjudication(ctx, LifeAdjudicationInput{
		ProfileID:          profile.ID,
		QuestID:            quest.ID,
		Status:             "suggested",
		Verdict:            "partial",
		Reason:             "Needs one more proof item.",
		SuggestedExp:       40,
		SuggestedGold:      10,
		SuggestedNextSteps: []string{"Attach the diff"},
		EvidenceSnapshot:   []map[string]any{{"summary": evidence.Summary}},
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	second, err := ls.CreateAdjudication(ctx, LifeAdjudicationInput{
		ProfileID:          profile.ID,
		QuestID:            quest.ID,
		Status:             "suggested",
		Verdict:            "completed",
		Reason:             "Evidence now shows the quest is done.",
		SuggestedExp:       150,
		SuggestedGold:      40,
		SuggestedNextSteps: []string{"Write a retro"},
		EvidenceSnapshot:   []map[string]any{{"summary": evidence.Summary}},
	})
	require.NoError(t, err)

	latest, err := ls.GetLatestAdjudicationByQuest(ctx, profile.ID, quest.ID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, second.Flag, latest.Flag)

	require.NoError(t, ls.MarkAdjudicationApplied(ctx, second.ID))
	applied, err := ls.GetAdjudicationByFlag(ctx, profile.ID, second.Flag)
	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.Equal(t, "applied", applied.Status)
	assert.NotNil(t, applied.AppliedAt)
	assert.Equal(t, "partial", first.Verdict)
}

func TestLifeStore_BatchQuestDMLookups(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "batch-dm-user", "Ada", "Architect")
	require.NoError(t, err)
	characteristic, err := ls.CreateCharacteristic(ctx, profile.ID, "INT", "Intelligence")
	require.NoError(t, err)
	skill, err := ls.CreateSkill(ctx, profile.ID, characteristic.ID, "Systems Design", 0.5)
	require.NoError(t, err)

	q1, err := ls.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID: profile.ID, SkillID: skill.ID, Title: "Q1", Prompt: "p1",
		Type: "One-Time", AiEvaluatedDifficulty: "A", BaseExpReward: 10, BaseGoldReward: 1, DropTier: "Epic",
	})
	require.NoError(t, err)
	q2, err := ls.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID: profile.ID, SkillID: skill.ID, Title: "Q2", Prompt: "p2",
		Type: "One-Time", AiEvaluatedDifficulty: "B", BaseExpReward: 20, BaseGoldReward: 2, DropTier: "Rare",
	})
	require.NoError(t, err)

	_, err = ls.CreateEvidence(ctx, LifeEvidenceInput{
		ProfileID: profile.ID, QuestID: &q1.ID, SourceType: "note", Content: "e1", Summary: "e1",
	})
	require.NoError(t, err)
	_, err = ls.CreateEvidence(ctx, LifeEvidenceInput{
		ProfileID: profile.ID, QuestID: &q2.ID, SourceType: "note", Content: "e2", Summary: "e2",
	})
	require.NoError(t, err)

	_, err = ls.CreateAdjudication(ctx, LifeAdjudicationInput{
		ProfileID: profile.ID, QuestID: q1.ID, Status: "suggested", Verdict: "partial", Reason: "old",
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	second, err := ls.CreateAdjudication(ctx, LifeAdjudicationInput{
		ProfileID: profile.ID, QuestID: q1.ID, Status: "suggested", Verdict: "completed", Reason: "new",
	})
	require.NoError(t, err)

	evidence, err := ls.ListEvidenceByQuestIDs(ctx, profile.ID, []int64{q1.ID, q2.ID})
	require.NoError(t, err)
	require.Len(t, evidence, 2)

	latest, err := ls.MapLatestAdjudicationsByQuestIDs(ctx, profile.ID, []int64{q1.ID, q2.ID})
	require.NoError(t, err)
	require.Len(t, latest, 1)
	assert.Equal(t, second.Flag, latest[q1.ID].Flag)
	assert.Nil(t, latest[q2.ID])
}

func TestLifeStore_PersistFailQuestRustBatch(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "fail-batch-user", "Ada", "Architect")
	require.NoError(t, err)
	_, err = ls.EnsureEquippedSlots(ctx, profile.ID)
	require.NoError(t, err)
	characteristic, err := ls.CreateCharacteristic(ctx, profile.ID, "INT", "Intelligence")
	require.NoError(t, err)
	skill, err := ls.CreateSkill(ctx, profile.ID, characteristic.ID, "Systems Design", 0.5)
	require.NoError(t, err)
	eq, err := ls.UpsertEquipment(ctx, "eq-flag", "Sword", "Common", "weapon", "", map[string]any{"atk": 1}, nil)
	require.NoError(t, err)
	inv1, err := ls.CreateInventory(ctx, profile.ID, eq.ID, nil, "none")
	require.NoError(t, err)
	inv2, err := ls.CreateInventory(ctx, profile.ID, eq.ID, nil, "none")
	require.NoError(t, err)
	quest, err := ls.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID: profile.ID, SkillID: skill.ID, Title: "Fail me", Prompt: "p",
		Type: "One-Time", AiEvaluatedDifficulty: "A", BaseExpReward: 10, BaseGoldReward: 1, DropTier: "Common",
	})
	require.NoError(t, err)

	until := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	require.NoError(t, ls.PersistFailQuest(ctx, profile.ID, quest.ID, []int64{inv1.ID, inv2.ID}, until))

	failed, err := ls.GetQuest(ctx, quest.ID)
	require.NoError(t, err)
	assert.Equal(t, "Failed", failed.Status)

	invByID, err := ls.MapInventoryByIDs(ctx, []int64{inv1.ID, inv2.ID})
	require.NoError(t, err)
	require.NotNil(t, invByID[inv1.ID].TarnishedUntil)
	require.NotNil(t, invByID[inv2.ID].TarnishedUntil)
	assert.WithinDuration(t, until, *invByID[inv1.ID].TarnishedUntil, time.Second)
}
