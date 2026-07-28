package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var lifeWebserviceRules = []webservice.Rule{
	webservice.Get("/life", lifeDashboardPage, route.WithNotAuth()),
	webservice.Get("/life/character", lifeCharacterPage, route.WithNotAuth()),
	webservice.Post("/life/character/class", lifeSetClass, route.WithNotAuth()),
	webservice.Post("/life/character/plan", lifeCreatePlanNode, route.WithNotAuth()),
	webservice.Post("/life/character/plan/:flag/confirm-habit", lifeConfirmHabit, route.WithNotAuth()),
	webservice.Post("/life/character/plan/breakdown/preview", lifePreviewBreakdown, route.WithNotAuth()),
	webservice.Post("/life/character/plan/breakdown/import", lifeImportBreakdown, route.WithNotAuth()),
	webservice.Post("/life/goals", lifeCreateGoal, route.WithNotAuth()),
	webservice.Post("/life/goals/:flag", lifeUpdateGoal, route.WithNotAuth()),
	webservice.Post("/life/goals/:flag/status", lifeSetGoalStatus, route.WithNotAuth()),
	webservice.Post("/life/goals/:flag/delete", lifeDeleteGoal, route.WithNotAuth()),
	webservice.Get("/life/quests", lifeQuestsPage, route.WithNotAuth()),
	webservice.Post("/life/quests", lifeCreateQuest, route.WithNotAuth()),
	webservice.Post("/life/quests/:flag/complete", lifeCompleteQuest, route.WithNotAuth()),
	webservice.Post("/life/quests/:flag/fail", lifeFailQuest, route.WithNotAuth()),
	webservice.Post("/life/actions/:flag/complete", lifeCompleteActionOccurrence, route.WithNotAuth()),
	webservice.Post("/life/actions/:flag/skip", lifeSkipActionOccurrence, route.WithNotAuth()),
	webservice.Post("/life/habits/:flag/checkin", lifeCheckInHabit, route.WithNotAuth()),
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
	planTree := make([]pages.LifePlanNodeRow, 0, len(char.PlanTree))
	planParents := make([]pages.LifePlanParentOption, 0)
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
	for _, node := range char.PlanTree {
		planTree = append(planTree, mapLifePlanNodeRow(node))
		collectPlanParentOptions(node, 0, &planParents)
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
		PlanTree:        planTree,
		PlanParents:     planParents,
	}
}

func mapLifePlanNodeRow(node *lifemod.PlanNodeView) pages.LifePlanNodeRow {
	row := pages.LifePlanNodeRow{
		Flag:        node.Node.Flag,
		NodeType:    node.Node.NodeType,
		Title:       node.Node.Title,
		Description: node.Node.Description,
		Status:      node.Node.Status,
	}
	if node.Node.ParentID != nil {
		row.ParentFlag = fmt.Sprintf("%d", *node.Node.ParentID)
	}
	if node.Action != nil {
		row.TaskType = node.Action.TaskType
		row.TrackingMode = node.Action.TrackingMode
		row.SuggestedCadence = node.Action.SuggestedCadence
		row.NeedsConfirmation = node.Action.NeedsUserConfirmation
	}
	row.Children = make([]pages.LifePlanNodeRow, 0, len(node.Children))
	for _, child := range node.Children {
		row.Children = append(row.Children, mapLifePlanNodeRow(child))
	}
	return row
}

func mapLifeBreakdownSuggestionRow(node *lifecap.GoalBreakdownSuggestion) *pages.LifeBreakdownSuggestionRow {
	if node == nil {
		return nil
	}
	row := &pages.LifeBreakdownSuggestionRow{
		NodeType:         node.NodeType,
		Title:            node.Title,
		Description:      node.Description,
		SuggestedCadence: "",
	}
	if node.Action != nil {
		row.SuggestedCadence = node.Action.SuggestedCadence
		if node.Action.IsRepeatable {
			if strings.EqualFold(node.Action.TrackingMode, "consistency") {
				row.TaskType = "habit"
			} else {
				row.TaskType = "recurring"
			}
		} else {
			row.TaskType = "todo"
		}
	}
	row.Children = make([]pages.LifeBreakdownSuggestionRow, 0, len(node.Children))
	for _, child := range node.Children {
		childRow := mapLifeBreakdownSuggestionRow(&child)
		if childRow != nil {
			row.Children = append(row.Children, *childRow)
		}
	}
	return row
}

func collectPlanParentOptions(node *lifemod.PlanNodeView, depth int, out *[]pages.LifePlanParentOption) {
	if node == nil || node.Node == nil {
		return
	}
	*out = append(*out, pages.LifePlanParentOption{
		Flag:      node.Node.Flag,
		Label:     node.Node.Title,
		NodeType:  node.Node.NodeType,
		Depth:     depth,
		AllowText: planAllowedChildrenText(node.Node.NodeType),
	})
	for _, child := range node.Children {
		collectPlanParentOptions(child, depth+1, out)
	}
}

func planAllowedChildrenText(nodeType string) string {
	switch nodeType {
	case "goal":
		return "milestone, project"
	case "milestone":
		return "project"
	case "project":
		return "action"
	default:
		return ""
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

func lifeCreatePlanNode(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	nodeType := strings.TrimSpace(ctx.FormValue("node_type"))
	parentFlag := strings.TrimSpace(ctx.FormValue("parent_flag"))
	title := strings.TrimSpace(ctx.FormValue("title"))
	description := strings.TrimSpace(ctx.FormValue("description"))
	action := &lifemod.ActionInput{
		IsRepeatable:       strings.EqualFold(strings.TrimSpace(ctx.FormValue("is_repeatable")), "on"),
		TrackingMode:       strings.TrimSpace(ctx.FormValue("tracking_mode")),
		RepeatTrigger:      strings.TrimSpace(ctx.FormValue("repeat_trigger")),
		SuggestedCadence:   strings.TrimSpace(ctx.FormValue("suggested_cadence")),
		IsIdentityBuilding: strings.EqualFold(strings.TrimSpace(ctx.FormValue("is_identity_building")), "on"),
		Reason:             strings.TrimSpace(ctx.FormValue("reason")),
	}
	if nodeType != "action" {
		action = nil
	}
	if _, err := lifeService().CreatePlanNode(context.Background(), uid, parentFlag, nodeType, title, description, action); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/character")
	return ctx.SendStatus(http.StatusOK)
}

func lifeConfirmHabit(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().ConfirmHabitAction(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/character")
	return ctx.SendStatus(http.StatusOK)
}

func lifePreviewBreakdown(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	rootTitle := strings.TrimSpace(ctx.FormValue("root_title"))
	description := strings.TrimSpace(ctx.FormValue("description"))
	suggestion, err := lifeService().PreviewGoalBreakdown(context.Background(), uid, rootTitle, description)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	data, err := lifeIdentityData(uid, true)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	payload, err := sonic.MarshalString(suggestion)
	if err != nil {
		return toastError(ctx, "could not encode breakdown preview")
	}
	data.BreakdownPreview = &pages.LifeBreakdownPreviewData{
		RootTitle:   rootTitle,
		Description: description,
		PayloadJSON: payload,
		Tree:        mapLifeBreakdownSuggestionRow(suggestion),
	}
	ctx.Type("html")
	return pages.LifeCharacterPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}

func lifeImportBreakdown(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	payload := strings.TrimSpace(ctx.FormValue("payload_json"))
	if payload == "" {
		return toastError(ctx, "breakdown preview payload required")
	}
	var suggestion lifecap.GoalBreakdownSuggestion
	if err := sonic.UnmarshalString(payload, &suggestion); err != nil {
		return toastError(ctx, "invalid breakdown preview payload")
	}
	if err := lifeService().ImportGoalBreakdown(context.Background(), uid, &suggestion); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("Location", "/service/web/life/character")
	ctx.Status(http.StatusSeeOther)
	return ctx.SendString("redirect")
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
	today, err := svc.ListTodayActions(context.Background(), uid)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	logs, err := svc.ListActionLogs(context.Background(), uid, 20)
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	data := buildLifeQuestsData(svc, uid, char, pending, done, today, logs)
	ctx.Type("html")
	return pages.LifeQuestsPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}

func buildLifeQuestsData(svc *lifemod.Service, uid string, char *lifemod.CharacterView, pending, done []*gen.LifeQuest, today *lifemod.TodayBoardView, logs []lifemod.ActionLogView) pages.LifeQuestsData {
	goalRows := mapActiveGoalRows(char)
	rowsPending := mapPendingQuestRows(svc, uid, pending)
	rowsDone := mapCompletedQuestRows(done)
	logRows := mapActionLogRows(logs)
	todayActions := mapTodayActionRows(today, char.PlanTree)
	todayHabits := mapTodayHabitRows(today)
	return pages.LifeQuestsData{
		Pending: rowsPending, Completed: rowsDone, Goals: goalRows, TodayActions: todayActions, TodayHabits: todayHabits, ActionLogs: logRows,
		PendingCount: len(rowsPending),
	}
}

func mapActiveGoalRows(char *lifemod.CharacterView) []pages.LifeGoalRow {
	goalRows := make([]pages.LifeGoalRow, 0, len(char.Goals))
	for _, g := range char.Goals {
		if g.Status != "Active" {
			continue
		}
		goalRows = append(goalRows, pages.LifeGoalRow{Flag: g.Flag, Title: g.Title, Category: g.Category, Status: g.Status})
	}
	return goalRows
}

func mapPendingQuestRows(svc *lifemod.Service, uid string, pending []*gen.LifeQuest) []pages.LifeQuestRow {
	rows := make([]pages.LifeQuestRow, 0, len(pending))
	for _, q := range pending {
		chance, _ := svc.PreviewDropChance(context.Background(), uid, q.Flag)
		rows = append(rows, pages.LifeQuestRow{
			Flag: q.Flag, Title: q.Title, Prompt: q.Prompt, Type: q.Type, Difficulty: q.AiEvaluatedDifficulty,
			Fear: q.AiEvaluatedFear, Exp: q.BaseExpReward, Gold: q.BaseGoldReward, DropTier: q.DropTier,
			DropChance: chance, Status: q.Status,
		})
	}
	return rows
}

func mapCompletedQuestRows(done []*gen.LifeQuest) []pages.LifeQuestRow {
	rows := make([]pages.LifeQuestRow, 0, min(len(done), 20))
	for _, q := range done {
		if len(rows) >= 20 {
			break
		}
		rows = append(rows, pages.LifeQuestRow{
			Flag: q.Flag, Title: q.Title, Prompt: q.Prompt, Type: q.Type, Difficulty: q.AiEvaluatedDifficulty,
			Fear: q.AiEvaluatedFear, Exp: q.BaseExpReward, Gold: q.BaseGoldReward, DropTier: q.DropTier, Status: q.Status,
		})
	}
	return rows
}

func mapActionLogRows(logs []lifemod.ActionLogView) []pages.LifeActionLogRow {
	rows := make([]pages.LifeActionLogRow, 0, len(logs))
	for _, l := range logs {
		title := l.QuestTitle
		if title == "" {
			title = "Action"
		}
		rows = append(rows, pages.LifeActionLogRow{
			Flag: l.Flag, SourceType: l.SourceType, QuestTitle: title, GainedExp: l.GainedExp, GainedGold: l.GainedGold,
			Dice: l.Dice, HasDice: l.HasDice, Dropped: l.Dropped,
			When: l.CreatedAt.UTC().Format("2006-01-02 15:04"),
		})
	}
	return rows
}

func mapTodayActionRows(today *lifemod.TodayBoardView, planTree []*lifemod.PlanNodeView) []pages.LifeTodayActionRow {
	if today == nil {
		return nil
	}
	contextByFlag := make(map[string]string)
	descriptionByFlag := make(map[string]string)
	for _, node := range planTree {
		collectPlanNodeContext(node, nil, contextByFlag, descriptionByFlag)
	}
	rows := make([]pages.LifeTodayActionRow, 0, len(today.Actions))
	for _, action := range today.Actions {
		if action.Occurrence == nil || action.Node == nil {
			continue
		}
		row := pages.LifeTodayActionRow{
			Flag:        action.Occurrence.Flag,
			NodeFlag:    action.Node.Flag,
			Title:       action.Node.Title,
			ContextPath: contextByFlag[action.Node.Flag],
			Description: descriptionByFlag[action.Node.Flag],
			Kind:        action.Occurrence.Kind,
			State:       action.Occurrence.State,
			DueLabel:    action.Occurrence.DueAt.UTC().Format("2006-01-02"),
		}
		if action.Spec != nil {
			row.TaskType = action.Spec.TaskType
			row.TrackingMode = action.Spec.TrackingMode
			row.SuggestedCadence = action.Spec.SuggestedCadence
		}
		rows = append(rows, row)
	}
	return rows
}

func collectPlanNodeContext(node *lifemod.PlanNodeView, parents []string, contextByFlag, descriptionByFlag map[string]string) {
	if node == nil || node.Node == nil {
		return
	}
	if len(parents) > 0 {
		contextByFlag[node.Node.Flag] = strings.Join(parents, " / ")
	}
	descriptionByFlag[node.Node.Flag] = strings.TrimSpace(node.Node.Description)
	path := append(append([]string{}, parents...), node.Node.Title)
	for _, child := range node.Children {
		collectPlanNodeContext(child, path, contextByFlag, descriptionByFlag)
	}
}

func mapTodayHabitRows(today *lifemod.TodayBoardView) []pages.LifeTodayHabitRow {
	if today == nil {
		return nil
	}
	rows := make([]pages.LifeTodayHabitRow, 0, len(today.Habits))
	for _, habit := range today.Habits {
		if habit.Node == nil {
			continue
		}
		row := pages.LifeTodayHabitRow{
			NodeFlag:  habit.Node.Flag,
			Title:     habit.Node.Title,
			CheckedIn: habit.CheckedIn,
		}
		if habit.Spec != nil {
			row.TaskType = habit.Spec.TaskType
			row.SuggestedCadence = habit.Spec.SuggestedCadence
		}
		rows = append(rows, row)
	}
	return rows
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

func lifeCompleteActionOccurrence(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().CompleteActionOccurrence(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeSkipActionOccurrence(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().SkipActionOccurrence(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeCheckInHabit(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().CheckInHabit(context.Background(), uid, ctx.Params("flag"), time.Now().UTC()); err != nil {
		return toastError(ctx, lifeUserError(err))
	}
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
			SlotField: slotField,
			BuffText:  pages.LifeFormatBuffText(it.Equip.StatBuffs),
			PerkText:  pages.LifeFormatPerkText(it.Equip.AiUnlockedPrivilege),
			Lore:      lore,
			LoreStatus: it.Inv.LoreStatus,
			Equipped: slotField != "", Tarnished: pkglife.IsTarnished(it.Inv.TarnishedUntil, time.Now()),
		})
	}
	pending, err := lifeService().ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return toastError(ctx, lifeUserError(err))
	}
	ctx.Type("html")
	return pages.LifeInventoryPage(pages.LifeInventoryData{
		Slots: pages.LifeBuildEquipSlots(rows), Items: rows, PendingCount: len(pending),
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
