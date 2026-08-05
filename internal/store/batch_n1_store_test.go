package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/agenttodo"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestAgentStore_ReplaceAndMergeTodos(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	as := NewAgentStore(client)
	ctx := context.Background()

	require.NoError(t, as.ReplaceAgentTodosForSession(ctx, "sess-1", []*gen.AgentTodo{
		{Flag: types.Id(), ItemID: "a", Content: "one", Status: "pending", SortOrder: 1},
		{Flag: types.Id(), ItemID: "b", Content: "two", Status: "pending", SortOrder: 2},
	}))
	rows, err := client.AgentTodo.Query().Where(agenttodo.SessionIDEQ("sess-1")).Order(gen.Asc(agenttodo.FieldSortOrder)).All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "one", rows[0].Content)

	require.NoError(t, as.MergeAgentTodosForSession(ctx, "sess-1", []*gen.AgentTodo{
		{Flag: types.Id(), ItemID: "a", Content: "one-updated", Status: "done", SortOrder: 1},
		{Flag: types.Id(), ItemID: "c", Content: "three", Status: "pending", SortOrder: 3},
	}))
	rows, err = client.AgentTodo.Query().Where(agenttodo.SessionIDEQ("sess-1")).Order(gen.Asc(agenttodo.FieldSortOrder)).All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	byItem := map[string]*gen.AgentTodo{}
	for _, row := range rows {
		byItem[row.ItemID] = row
	}
	assert.Equal(t, "one-updated", byItem["a"].Content)
	assert.Equal(t, "done", byItem["a"].Status)
	assert.Equal(t, "two", byItem["b"].Content)
	assert.Equal(t, "three", byItem["c"].Content)

	require.NoError(t, as.ReplaceAgentTodosForSession(ctx, "sess-1", []*gen.AgentTodo{
		{Flag: types.Id(), ItemID: "z", Content: "only", Status: "pending", SortOrder: 0},
	}))
	rows, err = client.AgentTodo.Query().Where(agenttodo.SessionIDEQ("sess-1")).All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "z", rows[0].ItemID)
}

func TestHubStore_SaveHomelabApps(t *testing.T) {
	t.Parallel()
	client := sqlitetest.OpenClient(t, t.Name())
	hs := NewHubStore(client)
	ctx := context.Background()

	require.NoError(t, hs.SaveHomelabApps(ctx, []homelab.App{
		{Name: "karakeep", Path: "/apps/karakeep", Status: homelab.AppStatusRunning},
		{Name: "archivebox", Path: "/apps/archivebox", Status: homelab.AppStatusStopped},
	}))
	infos, err := hs.ListApps(ctx)
	require.NoError(t, err)
	require.Len(t, infos, 2)

	require.NoError(t, hs.SaveHomelabApps(ctx, []homelab.App{
		{Name: "karakeep", Path: "/apps/karakeep-v2", Status: homelab.AppStatusPartial},
		{Name: "immich", Path: "/apps/immich", Status: homelab.AppStatusRunning},
	}))
	infos, err = hs.ListApps(ctx)
	require.NoError(t, err)
	require.Len(t, infos, 3)

	row, err := client.App.Query().All(ctx)
	require.NoError(t, err)
	byName := map[string]*gen.App{}
	for _, r := range row {
		byName[r.Name] = r
	}
	assert.Equal(t, "/apps/karakeep-v2", byName["karakeep"].Path)
	assert.Equal(t, string(homelab.AppStatusPartial), byName["karakeep"].Status)
	assert.Equal(t, "/apps/immich", byName["immich"].Path)
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

	first, err := ls.CreateAdjudication(ctx, LifeAdjudicationInput{
		ProfileID: profile.ID, QuestID: q1.ID, Status: "suggested", Verdict: "partial", Reason: "old",
	})
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	second, err := ls.CreateAdjudication(ctx, LifeAdjudicationInput{
		ProfileID: profile.ID, QuestID: q1.ID, Status: "suggested", Verdict: "completed", Reason: "new",
	})
	require.NoError(t, err)
	_ = first

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
