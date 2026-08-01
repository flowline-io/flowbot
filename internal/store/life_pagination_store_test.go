package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestLifeStore_ListQuestsPage(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "quests-page-user", "Ada", "Architect")
	require.NoError(t, err)
	characteristic, err := ls.CreateCharacteristic(ctx, profile.ID, "INT", "Intelligence")
	require.NoError(t, err)
	skill, err := ls.CreateSkill(ctx, profile.ID, characteristic.ID, "Systems Design", 0.5)
	require.NoError(t, err)

	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i := range 5 {
		quest, err := ls.CreateQuest(ctx, &gen.LifeQuest{
			LifeProfileID:         profile.ID,
			SkillID:               skill.ID,
			Title:                 fmt.Sprintf("Quest %d", i),
			Prompt:                "done",
			Type:                  "One-Time",
			AiEvaluatedDifficulty: "B",
			BaseExpReward:         10,
			BaseGoldReward:        5,
			DropTier:              "Common",
		})
		require.NoError(t, err)
		completedAt := base.Add(time.Duration(i) * time.Minute)
		_, err = client.LifeQuest.UpdateOneID(quest.ID).
			SetStatus("Completed").
			SetCompletedAt(completedAt).
			Save(ctx)
		require.NoError(t, err)
	}
	_, err = ls.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "Still pending",
		Prompt:                "todo",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "B",
		BaseExpReward:         10,
		BaseGoldReward:        5,
		DropTier:              "Common",
	})
	require.NoError(t, err)

	page1, total, err := ls.ListQuestsPage(ctx, profile.ID, "Completed", 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page1, 2)
	assert.Equal(t, "Quest 4", page1[0].Title)
	assert.Equal(t, "Quest 3", page1[1].Title)

	page2, total, err := ls.ListQuestsPage(ctx, profile.ID, "Completed", 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page2, 2)
	assert.Equal(t, "Quest 2", page2[0].Title)
	assert.Equal(t, "Quest 1", page2[1].Title)
}

func TestLifeStore_ListActionLogsPage(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	ls := NewLifeStore(client)
	ctx := context.Background()

	profile, err := ls.CreateProfile(ctx, "logs-page-user", "Ada", "Architect")
	require.NoError(t, err)

	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i := range 5 {
		createdAt := base.Add(time.Duration(i) * time.Minute)
		_, err := client.LifeActionLog.Create().
			SetFlag(types.Id()).
			SetLifeProfileID(profile.ID).
			SetSourceType("quest").
			SetSummary(fmt.Sprintf("Log %d", i)).
			SetGainedExp(i + 1).
			SetGainedGold(i).
			SetCreatedAt(createdAt).
			Save(ctx)
		require.NoError(t, err)
	}

	page1, total, err := ls.ListActionLogsPage(ctx, profile.ID, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page1, 2)
	assert.Equal(t, "Log 4", page1[0].Summary)
	assert.Equal(t, "Log 3", page1[1].Summary)

	page2, total, err := ls.ListActionLogsPage(ctx, profile.ID, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page2, 2)
	assert.Equal(t, "Log 2", page2[0].Summary)
	assert.Equal(t, "Log 1", page2[1].Summary)
}
