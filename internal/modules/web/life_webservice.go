package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var lifeWebserviceRules = []webservice.Rule{
	webservice.Get("/life", lifeDashboardPage, route.WithNotAuth()),
	webservice.Get("/life/character", lifeCharacterPage, route.WithNotAuth()),
	webservice.Post("/life/character/class", lifeSetClass, route.WithNotAuth()),
	webservice.Post("/life/goals", lifeCreateGoal, route.WithNotAuth()),
	webservice.Post("/life/goals/:flag", lifeUpdateGoal, route.WithNotAuth()),
	webservice.Post("/life/goals/:flag/status", lifeSetGoalStatus, route.WithNotAuth()),
	webservice.Post("/life/goals/:flag/delete", lifeDeleteGoal, route.WithNotAuth()),
	webservice.Get("/life/quests", lifeQuestsPage, route.WithNotAuth()),
	webservice.Post("/life/quests", lifeCreateQuest, route.WithNotAuth()),
	webservice.Post("/life/quests/:flag/complete", lifeCompleteQuest, route.WithNotAuth()),
	webservice.Post("/life/quests/:flag/fail", lifeFailQuest, route.WithNotAuth()),
	webservice.Get("/life/inventory", lifeInventoryPage, route.WithNotAuth()),
	webservice.Post("/life/inventory/:flag/equip", lifeEquipItem, route.WithNotAuth()),
	webservice.Post("/life/inventory/slots/:slot/unequip", lifeUnequipSlot, route.WithNotAuth()),
}

func lifeUserError(err error) string {
	if err == nil {
		return "Something went wrong"
	}
	msg := err.Error()
	const prefix = "life: "
	if !strings.HasPrefix(msg, prefix) {
		return "Could not complete that action. Please try again."
	}
	rest := strings.TrimPrefix(msg, prefix)
	switch {
	case strings.Contains(rest, "store not available"),
		strings.Contains(rest, "begin"),
		strings.Contains(rest, "commit"),
		strings.Contains(rest, "update "),
		strings.Contains(rest, "create "),
		strings.Contains(rest, "mark "),
		strings.Contains(rest, "append"),
		strings.Contains(rest, "lore"),
		strings.Contains(rest, "action log"),
		strings.Contains(rest, "daily respawn"),
		strings.Contains(rest, "equipment"),
		strings.Contains(rest, "skill missing"),
		strings.Contains(rest, "characteristic missing"),
		strings.Contains(rest, "service unavailable"):
		return "Could not complete that action. Please try again."
	default:
		return rest
	}
}

func lifeUID(ctx fiber.Ctx) (string, error) {
	uid, err := webUID(ctx)
	if err != nil {
		return "", err
	}
	return uid.String(), nil
}

func lifeIdentityData(uid string, showClassForm bool) (pages.LifeCharacterData, error) {
	char, err := lifeService().GetCharacter(context.Background(), uid)
	if err != nil {
		return pages.LifeCharacterData{}, err
	}
	pending, err := lifeService().ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return pages.LifeCharacterData{}, err
	}
	return mapLifeCharacterData(uid, char, len(pending), showClassForm), nil
}

func mapLifeCharacterData(uid string, char *lifemod.CharacterView, pendingCount int, showClassForm bool) pages.LifeCharacterData {
	stats := make([]pages.LifeStatRow, 0, len(char.Characteristics))
	for _, c := range char.Characteristics {
		stats = append(stats, pages.LifeBuildStatRow(c.Code, c.Name, c.Level, c.CurrentExp))
	}
	skills := make([]pages.LifeSkillRow, 0, len(char.Skills))
	for _, sk := range char.Skills {
		skills = append(skills, pages.LifeSkillRow{Name: sk.Name, Level: sk.Level, Exp: sk.CurrentExp})
	}
	goals := make([]pages.LifeGoalRow, 0, len(char.Goals))
	master, minor := "Set a Project goal", "Set an Area goal"
	for _, g := range char.Goals {
		goals = append(goals, pages.LifeGoalRow{Flag: g.Flag, Title: g.Title, Category: g.Category, Status: g.Status})
		if g.Status != "Active" {
			continue
		}
		switch g.Category {
		case "Project":
			if master == "Set a Project goal" {
				master = g.Title
			}
		case "Area":
			if minor == "Set an Area goal" {
				minor = g.Title
			}
		}
	}
	expNeed := pkglife.ExpToNextLevel(char.Profile.Level)
	levelProg := pages.LifeBuildStatRow("LVL", "Level", char.Profile.Level, char.Profile.Exp)
	hpCur, hpMax, heartsFilled, heartsTotal := pages.LifeHPFromStats(stats, char.Profile.Level)
	labelsJSON, valuesJSON := pages.LifeMarshalRadar(stats)
	return pages.LifeCharacterData{
		Nickname:        pages.LifeDisplayName(char.Profile.Nickname, uid),
		ClassType:       char.Profile.ClassType,
		Level:           char.Profile.Level,
		Exp:             char.Profile.Exp,
		ExpToNext:       expNeed,
		LevelFilledSegs: levelProg.FilledSegs,
		LevelTotalSegs:  levelProg.TotalSegs,
		Gold:            char.Profile.Gold,
		HPCurrent:       hpCur,
		HPMax:           hpMax,
		HeartsFilled:    heartsFilled,
		HeartsTotal:     heartsTotal,
		MasterObjective: master,
		MinorObjective:  minor,
		Strength:        pages.LifeClassStrength(char.Profile.ClassType),
		Weakness:        pages.LifeClassWeakness(char.Profile.ClassType),
		Characteristics: stats,
		Skills:          skills,
		DropRateBonus:   char.Buffs.DropRate,
		GoldMult:        char.Buffs.GoldMult,
		PendingCount:    pendingCount,
		RadarLabelsJSON: labelsJSON,
		RadarValuesJSON: valuesJSON,
		ShowClassForm:   showClassForm,
		Goals:           goals,
	}
}

func lifeDashboardPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	data, err := lifeIdentityData(uid, false)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Type("html")
	return pages.LifeDashboardPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}

func lifeCharacterPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	data, err := lifeIdentityData(uid, true)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Type("html")
	return pages.LifeCharacterPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}

func lifeSetClass(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	classType := strings.TrimSpace(ctx.FormValue("class_type"))
	if err := lifeService().SetClassType(context.Background(), uid, classType); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/character")
	return ctx.SendStatus(http.StatusOK)
}

func lifeCreateGoal(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	title := strings.TrimSpace(ctx.FormValue("title"))
	category := strings.TrimSpace(ctx.FormValue("category"))
	if _, err := lifeService().CreateGoal(context.Background(), uid, title, category); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/character")
	return ctx.SendStatus(http.StatusOK)
}

func lifeUpdateGoal(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	title := strings.TrimSpace(ctx.FormValue("title"))
	category := strings.TrimSpace(ctx.FormValue("category"))
	if err := lifeService().UpdateGoal(context.Background(), uid, ctx.Params("flag"), title, category); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/character")
	return ctx.SendStatus(http.StatusOK)
}

func lifeSetGoalStatus(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	status := strings.TrimSpace(ctx.FormValue("status"))
	if err := lifeService().SetGoalStatus(context.Background(), uid, ctx.Params("flag"), status); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/character")
	return ctx.SendStatus(http.StatusOK)
}

func lifeDeleteGoal(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().DeleteGoal(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/character")
	return ctx.SendStatus(http.StatusOK)
}

func lifeQuestsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	svc := lifeService()
	pending, err := svc.ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	done, err := svc.ListQuests(context.Background(), uid, "Completed")
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	char, err := svc.GetCharacter(context.Background(), uid)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	logs, err := svc.ListActionLogs(context.Background(), uid, 20)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	goalRows := make([]pages.LifeGoalRow, 0, len(char.Goals))
	for _, g := range char.Goals {
		if g.Status != "Active" {
			continue
		}
		goalRows = append(goalRows, pages.LifeGoalRow{Flag: g.Flag, Title: g.Title, Category: g.Category, Status: g.Status})
	}
	rowsPending := make([]pages.LifeQuestRow, 0, len(pending))
	for _, q := range pending {
		chance, _ := svc.PreviewDropChance(context.Background(), uid, q.Flag)
		rowsPending = append(rowsPending, pages.LifeQuestRow{
			Flag: q.Flag, Title: q.Title, Prompt: q.Prompt, Type: q.Type, Difficulty: q.AiEvaluatedDifficulty,
			Fear: q.AiEvaluatedFear, Exp: q.BaseExpReward, Gold: q.BaseGoldReward, DropTier: q.DropTier,
			DropChance: chance, Status: q.Status,
		})
	}
	rowsDone := make([]pages.LifeQuestRow, 0, len(done))
	for _, q := range done {
		if len(rowsDone) >= 20 {
			break
		}
		rowsDone = append(rowsDone, pages.LifeQuestRow{
			Flag: q.Flag, Title: q.Title, Prompt: q.Prompt, Type: q.Type, Difficulty: q.AiEvaluatedDifficulty,
			Fear: q.AiEvaluatedFear, Exp: q.BaseExpReward, Gold: q.BaseGoldReward, DropTier: q.DropTier, Status: q.Status,
		})
	}
	logRows := make([]pages.LifeActionLogRow, 0, len(logs))
	for _, l := range logs {
		title := l.QuestTitle
		if title == "" {
			title = "Quest"
		}
		logRows = append(logRows, pages.LifeActionLogRow{
			Flag: l.Flag, QuestTitle: title, GainedExp: l.GainedExp, GainedGold: l.GainedGold,
			Dice: l.Dice, HasDice: l.HasDice, Dropped: l.Dropped,
			When: l.CreatedAt.UTC().Format("2006-01-02 15:04"),
		})
	}
	data := pages.LifeQuestsData{
		Pending: rowsPending, Completed: rowsDone, Goals: goalRows, ActionLogs: logRows,
		PendingCount: len(rowsPending),
	}
	ctx.Type("html")
	return pages.LifeQuestsPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}

func lifeCreateQuest(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	prompt := strings.TrimSpace(ctx.FormValue("prompt"))
	if prompt == "" {
		return toastError(ctx, "prompt required")
	}
	goalFlag := strings.TrimSpace(ctx.FormValue("goal_flag"))
	q, _, chance, err := lifeService().CreateQuestFromPrompt(context.Background(), uid, prompt, goalFlag)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	msg := fmt.Sprintf("Quest rated %s — ~%.0f%% chance of %s loot", q.AiEvaluatedDifficulty, chance*100, q.DropTier)
	setShowToast(ctx, "success", msg)
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeCompleteQuest(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	flag := ctx.Params("flag")
	res, err := lifeService().CompleteQuest(context.Background(), uid, flag)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	msg := fmt.Sprintf("+%d EXP, +%d gold (dice %.2f)", res.GainedExp, res.GainedGold, res.Dice)
	if res.Dropped {
		msg = fmt.Sprintf("Loot drop! %s — +%d EXP, +%d gold", res.ItemName, res.GainedExp, res.GainedGold)
	}
	setShowToast(ctx, "success", msg)
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeFailQuest(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().FailQuest(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	setShowToast(ctx, "warning", "Quest failed — equipped gear tarnished for 24h")
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeInventoryPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	items, slots, err := lifeService().ListInventory(context.Background(), uid)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	equipped := map[int64]string{}
	if slots != nil {
		slotPairs := []struct {
			id    *int64
			field string
		}{
			{slots.HeadSlot, "head_slot"},
			{slots.WeaponSlot, "weapon_slot"},
			{slots.ArmorSlot, "armor_slot"},
			{slots.ShoesSlot, "shoes_slot"},
			{slots.AccessorySlot, "accessory_slot"},
			{slots.ArtifactSlot, "artifact_slot"},
		}
		for _, sp := range slotPairs {
			if sp.id != nil {
				equipped[*sp.id] = sp.field
			}
		}
	}
	rows := make([]pages.LifeInventoryRow, 0, len(items))
	for _, it := range items {
		name := it.Equip.Name
		if it.Inv.InstanceName != "" {
			name = it.Inv.InstanceName
		}
		lore := it.Equip.AiLoreText
		if it.Inv.InstanceLore != "" {
			lore = it.Inv.InstanceLore
		}
		slotField := equipped[it.Inv.ID]
		rows = append(rows, pages.LifeInventoryRow{
			Flag: it.Inv.Flag, Name: name, Rarity: it.Equip.Rarity, Slot: it.Equip.SlotType,
			SlotField: slotField, Lore: lore, LoreStatus: it.Inv.LoreStatus,
			Equipped: slotField != "", Tarnished: pkglife.IsTarnished(it.Inv.TarnishedUntil, time.Now()),
		})
	}
	pending, err := lifeService().ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Type("html")
	return pages.LifeInventoryPage(pages.LifeInventoryData{
		Items: rows, PendingCount: len(pending),
	}).Render(context.Background(), ctx.Response().BodyWriter())
}

func lifeEquipItem(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().Equip(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/inventory")
	return ctx.SendStatus(http.StatusOK)
}

func lifeUnequipSlot(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().Unequip(context.Background(), uid, ctx.Params("slot")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/inventory")
	return ctx.SendStatus(http.StatusOK)
}
