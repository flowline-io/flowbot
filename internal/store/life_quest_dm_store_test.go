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
