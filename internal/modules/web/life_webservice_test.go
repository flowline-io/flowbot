package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeplannode"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeprofile"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

func TestLifeImportBreakdownRefreshesPage(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		setWebEncryptor(nil)
		SetLifeService(nil)
	}()
	ls := store.NewLifeStore(client)
	SetLifeService(lifemod.NewService(ls))
	svc := lifeService()
	preflightSuggestion := &lifecap.GoalBreakdownSuggestion{
		NodeType:    "goal",
		Title:       "Ship AI DM",
		Description: "Root goal",
		Children: []lifecap.GoalBreakdownSuggestion{
			{
				NodeType: "project",
				Title:    "Prototype dialogue",
				Children: []lifecap.GoalBreakdownSuggestion{
					{
						NodeType: "action",
						Title:    "Write dialogue core",
						Action: &lifecap.GoalBreakdownActionSuggestion{
							IsRepeatable:  false,
							TrackingMode:  "completion",
							RepeatTrigger: "none",
							Reason:        "bootstrap",
						},
					},
				},
			},
		},
	}
	err := svc.ImportGoalBreakdown(context.Background(), "preflight-user", preflightSuggestion)
	require.NoError(t, err)

	form := url.Values{}
	form.Set("payload_json", `{"node_type":"goal","title":"Ship AI DM","description":"Root goal","children":[{"node_type":"project","title":"Prototype dialogue","children":[{"node_type":"action","title":"Write dialogue core","action":{"is_repeatable":false,"tracking_mode":"completion","repeat_trigger":"none","reason":"bootstrap"}}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/service/web/life/character/plan/breakdown/import", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addWebAuth(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	require.Equal(t, "/service/web/life/plan", resp.Header.Get("Location"))
	require.Equal(t, "redirect", string(body))

	profile, err := client.LifeProfile.Query().Where(lifeprofile.UserIDEQ("testuser")).Only(req.Context())
	require.NoError(t, err)
	count, err := client.LifePlanNode.Query().
		Where(lifeplannode.LifeProfileIDEQ(profile.ID), lifeplannode.TitleEQ("Ship AI DM")).
		Count(req.Context())
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestLifeSkillsPageRenders(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		setWebEncryptor(nil)
		SetLifeService(nil)
	}()
	ls := store.NewLifeStore(client)
	SetLifeService(lifemod.NewService(ls))

	req := httptest.NewRequest(http.MethodGet, "/service/web/life/skills", http.NoBody)
	addWebAuth(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, string(body), "Skill Tree")
	require.Contains(t, string(body), "Practice Evidence")
	require.Contains(t, string(body), "/service/web/life/skills")
}

func TestLifeSubmitQuestEvidenceRedirectsAndPersists(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		setWebEncryptor(nil)
		SetLifeService(nil)
	}()
	ls := store.NewLifeStore(client)
	SetLifeService(lifemod.NewService(ls))
	svc := lifeService()
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "testuser", "", "")
	require.NoError(t, err)
	chars, err := ls.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := ls.CreateSkill(ctx, profile.ID, chars[0].ID, "Systems Design", 0.5)
	require.NoError(t, err)
	quest, err := ls.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "Ship evidence flow",
		Prompt:                "Ship quest evidence",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "B",
		BaseExpReward:         80,
		BaseGoldReward:        25,
		DropTier:              "Rare",
	})
	require.NoError(t, err)

	form := url.Values{}
	form.Set("source_type", "note")
	form.Set("content", "Implemented the flow and wrote tests.")
	req := httptest.NewRequest(http.MethodPost, "/service/web/life/quests/"+quest.Flag+"/evidence", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addWebAuth(req)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "/service/web/life/quests", resp.Header.Get("HX-Redirect"))

	rows, err := ls.ListEvidenceByQuest(ctx, profile.ID, quest.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Contains(t, rows[0].Content, "wrote tests")
}

func TestLifeDismissQuestRedirectsAndClearsPending(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		setWebEncryptor(nil)
		SetLifeService(nil)
	}()
	ls := store.NewLifeStore(client)
	SetLifeService(lifemod.NewService(ls))
	svc := lifeService()
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "testuser", "", "")
	require.NoError(t, err)
	chars, err := ls.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := ls.CreateSkill(ctx, profile.ID, chars[0].ID, "Focus", 0.5)
	require.NoError(t, err)
	quest, err := ls.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "Dismiss me",
		Prompt:                "Clear stuck quest",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "E",
		BaseExpReward:         10,
		BaseGoldReward:        3,
		DropTier:              "Common",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/service/web/life/quests/"+quest.Flag+"/dismiss", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "/service/web/life/quests", resp.Header.Get("HX-Redirect"))

	got, err := ls.GetQuestByFlag(ctx, profile.ID, quest.Flag)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "Dismissed", got.Status)
}

func TestLifeQuestsPageShowsQuestDMControls(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		setWebEncryptor(nil)
		SetLifeService(nil)
	}()
	ls := store.NewLifeStore(client)
	SetLifeService(lifemod.NewService(ls))
	svc := lifeService()
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "testuser", "", "")
	require.NoError(t, err)
	chars, err := ls.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := ls.CreateSkill(ctx, profile.ID, chars[0].ID, "Systems Design", 0.5)
	require.NoError(t, err)
	_, err = ls.CreateQuest(ctx, &gen.LifeQuest{
		LifeProfileID:         profile.ID,
		SkillID:               skill.ID,
		Title:                 "Ship evidence flow",
		Prompt:                "Ship quest evidence",
		Type:                  "One-Time",
		AiEvaluatedDifficulty: "B",
		BaseExpReward:         80,
		BaseGoldReward:        25,
		DropTier:              "Rare",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/service/web/life/quests", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, string(body), "Submit evidence")
	require.Contains(t, string(body), "Request ruling")
	require.Contains(t, string(body), "Dismiss")
	require.Contains(t, string(body), `data-testid="life-quest-dismiss-`)
}

func TestLifeQuestsPagePaginatesCompletedAndActionLogs(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		setWebEncryptor(nil)
		SetLifeService(nil)
	}()
	ls := store.NewLifeStore(client)
	SetLifeService(lifemod.NewService(ls))
	svc := lifeService()
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "testuser", "", "")
	require.NoError(t, err)
	chars, err := ls.ListCharacteristics(ctx, profile.ID)
	require.NoError(t, err)
	require.NotEmpty(t, chars)
	skill, err := ls.CreateSkill(ctx, profile.ID, chars[0].ID, "Focus", 0.5)
	require.NoError(t, err)

	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	total := pages.LifeDefaultListPerPage + 1
	for i := range total {
		quest, err := ls.CreateQuest(ctx, &gen.LifeQuest{
			LifeProfileID:         profile.ID,
			SkillID:               skill.ID,
			Title:                 fmt.Sprintf("Completed %02d", i),
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
			SetSummary(fmt.Sprintf("Log %02d", i)).
			SetGainedExp(10).
			SetGainedGold(5).
			SetCreatedAt(base.Add(time.Duration(i) * time.Minute)).
			Save(ctx)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/service/web/life/quests?completed_page=2&logs_page=2&history_tab=logs", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	html := string(body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, html, `id="life-history"`)
	require.Contains(t, html, `data-testid="life-history-tabs"`)
	require.Contains(t, html, `data-testid="life-history-tab-logs"`)
	require.Contains(t, html, "Completed 00")
	require.NotContains(t, html, "Completed 10")
	require.Contains(t, html, `data-testid="life-action-logs-pager"`)
	require.Contains(t, html, `href="/service/web/life/quests?completed_page=2&amp;history_tab=logs#life-history"`)
	require.Contains(t, html, `href="/service/web/life/quests?completed_page=2&amp;logs_page=2#life-history"`)
}

func TestLifeRewardsPagePaginatesArchiveTabs(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		setWebEncryptor(nil)
		SetLifeService(nil)
	}()
	ls := store.NewLifeStore(client)
	SetLifeService(lifemod.NewService(ls))
	svc := lifeService()
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "testuser", "", "")
	require.NoError(t, err)
	require.NoError(t, ls.SetProfileGold(ctx, profile.ID, 1000))

	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	total := pages.LifeDefaultListPerPage + 1
	for i := range total {
		reward, err := svc.CreateReward(ctx, "testuser", lifemod.CreateRewardInput{
			Name: fmt.Sprintf("Archive %02d", i), Price: 10,
		})
		require.NoError(t, err)
		require.NoError(t, svc.DeactivateReward(ctx, "testuser", reward.Flag))
		_, err = ls.CreateRewardRedemption(ctx, profile.ID, reward.ID, reward.Name, 10, base.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/service/web/life/rewards?redemptions_page=2&inactive_page=2&archive_tab=deactivated", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	html := string(body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, html, `id="life-rewards-archive"`)
	require.Contains(t, html, `data-testid="life-rewards-archive-tabs"`)
	require.Contains(t, html, `data-testid="life-rewards-tab-deactivated"`)
	require.Contains(t, html, "Archive 00")
	require.NotContains(t, html, "Archive 10")
	require.Contains(t, html, `data-testid="life-inactive-pager"`)
	require.Contains(t, html, `href="/service/web/life/rewards?archive_tab=deactivated&amp;redemptions_page=2#life-rewards-archive"`)
	require.Contains(t, html, `href="/service/web/life/rewards?inactive_page=2&amp;redemptions_page=2#life-rewards-archive"`)
}

func TestLifeInventoryPagePaginatesBackpack(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	defer func() {
		store.Database = nil
		handler = moduleHandler{}
		config = configType{}
		setWebEncryptor(nil)
		SetLifeService(nil)
	}()
	ls := store.NewLifeStore(client)
	SetLifeService(lifemod.NewService(ls))
	svc := lifeService()
	ctx := context.Background()

	profile, err := svc.EnsureProfile(ctx, "testuser", "", "")
	require.NoError(t, err)
	total := pages.LifeDefaultListPerPage + 1
	var firstFlag string
	for i := range total {
		eq, err := ls.UpsertEquipment(ctx, fmt.Sprintf("web-eq-%d", i), fmt.Sprintf("Pack %02d", i), "Common", "Armor", "", nil, nil)
		require.NoError(t, err)
		inv, err := ls.CreateInventory(ctx, profile.ID, eq.ID, nil, "none")
		require.NoError(t, err)
		if i == 0 {
			firstFlag = inv.Flag
		}
	}
	require.NoError(t, svc.Equip(ctx, "testuser", firstFlag))

	req := httptest.NewRequest(http.MethodGet, "/service/web/life/inventory?backpack_page=2", http.NoBody)
	addWebAuth(req)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	html := string(body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, html, `id="life-backpack"`)
	require.Contains(t, html, `data-testid="life-backpack-pager"`)
	require.Contains(t, html, "Pack 00")
	require.NotContains(t, html, "Pack 10")
	require.Contains(t, html, `href="/service/web/life/inventory#life-backpack"`)
	require.Contains(t, html, `data-testid="life-equip-slot-armor"`)
	require.Contains(t, html, "Pack 00") // equipped board still shows oldest item when paging
}
