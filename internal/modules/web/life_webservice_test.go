package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeplannode"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/lifeprofile"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
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
