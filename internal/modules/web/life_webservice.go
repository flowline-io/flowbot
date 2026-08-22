package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
	"github.com/flowline-io/flowbot/pkg/flog"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var lifeWebserviceRules = []webservice.Rule{
	webservice.Get("/life", lifeDashboardPage),
	webservice.Get("/life/stats", lifeStatsPage),
	webservice.Get("/life/stats/panel", lifeStatsPanel),
	webservice.Get("/life/character", lifeCharacterPage),
	webservice.Get("/life/goals", lifeGoalsPage),
	webservice.Get("/life/plan", lifePlanPage),
	webservice.Get("/life/skills", lifeSkillsPage),
	webservice.Post("/life/character/class", lifeSetClass),
	webservice.Post("/life/character/plan", lifeCreatePlanNode),
	webservice.Post("/life/character/plan/:flag/confirm-habit", lifeConfirmHabit),
	webservice.Post("/life/character/plan/breakdown/preview", lifePreviewBreakdown),
	webservice.Post("/life/character/plan/breakdown/import", lifeImportBreakdown),
	webservice.Post("/life/goals", lifeCreateGoal),
	webservice.Post("/life/goals/:flag", lifeUpdateGoal),
	webservice.Post("/life/goals/:flag/status", lifeSetGoalStatus),
	webservice.Post("/life/goals/:flag/delete", lifeDeleteGoal),
	webservice.Get("/life/quests", lifeQuestsPage),
	webservice.Post("/life/quests", lifeCreateQuest),
	webservice.Post("/life/quests/:flag/evidence", lifeSubmitQuestEvidence),
	webservice.Post("/life/quests/:flag/adjudicate", lifeAdjudicateQuest),
	webservice.Post("/life/quests/:flag/adjudication/:adjudicationFlag/apply", lifeApplyQuestAdjudication),
	webservice.Post("/life/quests/:flag/complete", lifeCompleteQuest),
	webservice.Post("/life/quests/:flag/fail", lifeFailQuest),
	webservice.Post("/life/quests/:flag/dismiss", lifeDismissQuest),
	webservice.Post("/life/actions/:flag/complete", lifeCompleteActionOccurrence),
	webservice.Post("/life/actions/:flag/skip", lifeSkipActionOccurrence),
	webservice.Post("/life/habits/:flag/checkin", lifeCheckInHabit),
	webservice.Get("/life/inventory", lifeInventoryPage),
	webservice.Get("/life/achievements", lifeAchievementsPage),
	webservice.Get("/life/rewards", lifeRewardsPage),
	webservice.Post("/life/rewards", lifeCreateReward),
	webservice.Post("/life/rewards/:flag", lifeUpdateReward),
	webservice.Post("/life/rewards/:flag/deactivate", lifeDeactivateReward),
	webservice.Post("/life/rewards/:flag/restore", lifeRestoreReward),
	webservice.Post("/life/rewards/:flag/redeem", lifeRedeemReward),
	webservice.Post("/life/inventory/:flag/equip", lifeEquipItem),
	webservice.Post("/life/inventory/slots/:slot/unequip", lifeUnequipSlot),
}

func lifeUserError(c fiber.Ctx, err error) string {
	if err == nil {
		return webMsg(c, "toast.life.generic_error")
	}
	if errors.Is(err, types.ErrProvider) ||
		errors.Is(err, types.ErrUnavailable) ||
		errors.Is(err, types.ErrInternal) ||
		errors.Is(err, types.ErrTimeout) {
		return webMsg(c, "toast.life.action_failed")
	}
	msg := err.Error()
	const prefix = "life: "
	if !strings.HasPrefix(msg, prefix) {
		return webMsg(c, "toast.life.action_failed")
	}
	rest := strings.TrimPrefix(msg, prefix)
	if errors.Is(err, types.ErrNotFound) ||
		errors.Is(err, types.ErrInvalidArgument) ||
		errors.Is(err, types.ErrConflict) {
		return rest
	}
	return webMsg(c, "toast.life.action_failed")
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
	goals := mapLifeGoalRows(char.Goals)
	planTree := make([]pages.LifePlanNodeRow, 0, len(char.PlanTree))
	planParents := make([]pages.LifePlanParentOption, 0)
	master, minor := "Set a Project goal", "Set an Area goal"
	for _, g := range goals {
		if g.Status != pkglife.GoalStatusActive {
			continue
		}
		switch g.Category {
		case pkglife.GoalCategoryProject:
			if master == "Set a Project goal" {
				master = g.Title
			}
		case pkglife.GoalCategoryArea:
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
		row.Difficulty = node.Action.Difficulty
		row.Exp = node.Action.BaseExpReward
		row.Gold = node.Action.BaseGoldReward
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
		row.Difficulty = node.Action.Difficulty
		row.Exp = node.Action.BaseExp
		row.Gold = node.Action.BaseGold
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Type("html")
	return pages.LifeDashboardPage(data).Render(ctx.Context(), ctx.Response().BodyWriter())
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Type("html")
	return pages.LifeCharacterPage(data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func lifeGoalsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	data, err := lifeGoalsPageData(uid)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Type("html")
	return pages.LifeGoalsPage(data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func lifePlanPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	data, err := lifePlanPageData(uid)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Type("html")
	return pages.LifePlanPage(data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func lifeGoalsPageData(uid string) (pages.LifeGoalsData, error) {
	identity, err := lifeIdentityData(uid, false)
	if err != nil {
		return pages.LifeGoalsData{}, err
	}
	activeAreas := make([]pages.LifeGoalRow, 0)
	for _, g := range identity.Goals {
		if g.Category == pkglife.GoalCategoryArea && g.Status == pkglife.GoalStatusActive {
			activeAreas = append(activeAreas, g)
		}
	}
	return pages.LifeGoalsData{
		Goals:         identity.Goals,
		Groups:        pages.LifeGroupGoals(identity.Goals),
		ActiveAreas:   activeAreas,
		DropRateBonus: identity.DropRateBonus,
		GoldMult:      identity.GoldMult,
		PendingCount:  identity.PendingCount,
	}, nil
}

func lifePlanPageData(uid string) (pages.LifePlanData, error) {
	identity, err := lifeIdentityData(uid, false)
	if err != nil {
		return pages.LifePlanData{}, err
	}
	return pages.LifePlanData{
		PlanTree:     identity.PlanTree,
		PlanParents:  identity.PlanParents,
		PendingCount: identity.PendingCount,
	}, nil
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
		return toastError(ctx, lifeUserError(ctx, err))
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
		TaskType:           strings.TrimSpace(ctx.FormValue("task_type")),
		IsRepeatable:       strings.EqualFold(strings.TrimSpace(ctx.FormValue("is_repeatable")), "on"),
		TrackingMode:       strings.TrimSpace(ctx.FormValue("tracking_mode")),
		RepeatTrigger:      strings.TrimSpace(ctx.FormValue("repeat_trigger")),
		SuggestedCadence:   strings.TrimSpace(ctx.FormValue("suggested_cadence")),
		IsIdentityBuilding: strings.EqualFold(strings.TrimSpace(ctx.FormValue("is_identity_building")), "on"),
		Reason:             strings.TrimSpace(ctx.FormValue("reason")),
		DependencyFlags:    parseLifeDependencyFlags(ctx.FormValue("dependency_flags")),
	}
	if nodeType != "action" {
		action = nil
	}
	if _, err := lifeService().CreatePlanNode(context.Background(), uid, parentFlag, nodeType, title, description, action); err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/plan")
	return ctx.SendStatus(http.StatusOK)
}

func parseLifeDependencyFlags(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/plan")
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	data, err := lifePlanPageData(uid)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	payload, err := sonic.MarshalString(suggestion)
	if err != nil {
		return toastErrorKey(ctx, "toast.life.breakdown_encode_failed")
	}
	data.BreakdownPreview = &pages.LifeBreakdownPreviewData{
		RootTitle:   rootTitle,
		Description: description,
		PayloadJSON: payload,
		Tree:        mapLifeBreakdownSuggestionRow(suggestion),
	}
	ctx.Type("html")
	return pages.LifePlanPage(data).Render(ctx.Context(), ctx.Response().BodyWriter())
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
		return toastErrorKey(ctx, "toast.life.breakdown_payload_required")
	}
	var suggestion lifecap.GoalBreakdownSuggestion
	if err := sonic.UnmarshalString(payload, &suggestion); err != nil {
		return toastErrorKey(ctx, "toast.life.breakdown_payload_invalid")
	}
	if err := lifeService().ImportGoalBreakdown(context.Background(), uid, &suggestion); err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Set("Location", "/service/web/life/plan")
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
	areaFlag := strings.TrimSpace(ctx.FormValue("area_flag"))
	if _, err := lifeService().CreateGoal(context.Background(), uid, title, category, areaFlag); err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/goals")
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
	areaFlag := strings.TrimSpace(ctx.FormValue("area_flag"))
	if err := lifeService().UpdateGoal(context.Background(), uid, ctx.Params("flag"), title, category, areaFlag); err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/goals")
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/goals")
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/goals")
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
	completedPage := parsePositiveIntQuery(ctx, "completed_page", 1)
	logsPage := parsePositiveIntQuery(ctx, "logs_page", 1)
	historyTab := pages.LifeNormalizeHistoryTab(ctx.Query("history_tab"))
	svc := lifeService()
	pending, err := svc.ListPendingQuestDMViews(context.Background(), uid)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	done, doneTotal, err := svc.ListCompletedQuestsPage(context.Background(), uid, completedPage, pages.LifeDefaultListPerPage)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	char, err := svc.GetCharacter(context.Background(), uid)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	today, err := svc.ListTodayActions(context.Background(), uid)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	logs, logsTotal, err := svc.ListActionLogsPage(context.Background(), uid, logsPage, pages.LifeDefaultListPerPage)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	data := buildLifeQuestsData(char, pending, done, today, logs, completedPage, doneTotal, logsPage, logsTotal, historyTab)
	ctx.Type("html")
	return pages.LifeQuestsPage(data).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func parsePositiveIntQuery(ctx fiber.Ctx, key string, fallback int) int {
	raw := strings.TrimSpace(ctx.Query(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return fallback
	}
	return v
}

func buildLifeQuestsData(
	char *lifemod.CharacterView,
	pending []lifemod.QuestDMView,
	done []*gen.LifeQuest,
	today *lifemod.TodayBoardView,
	logs []lifemod.ActionLogView,
	completedPage, doneTotal, logsPage, logsTotal int,
	historyTab string,
) pages.LifeQuestsData {
	goalRows := mapActiveGoalRows(char)
	rowsPending := mapPendingQuestRows(pending)
	rowsDone := mapCompletedQuestRows(done)
	logRows := mapActionLogRows(logs)
	todayActions := mapTodayActionRows(today, char.PlanTree)
	todayHabits := mapTodayHabitRows(today)
	completedInfo := pages.LifeWithCompletedPager(
		pages.LifeBuildPageInfo(completedPage, pages.LifeDefaultListPerPage, doneTotal),
		logsPage,
	)
	logsInfo := pages.LifeWithActionLogsPager(
		pages.LifeBuildPageInfo(logsPage, pages.LifeDefaultListPerPage, logsTotal),
		completedPage,
	)
	return pages.LifeQuestsData{
		Pending: rowsPending, Completed: rowsDone, Goals: goalRows, TodayActions: todayActions, TodayHabits: todayHabits, ActionLogs: logRows,
		CompletedPage: completedInfo, ActionLogsPage: logsInfo,
		HistoryTab:   pages.LifeNormalizeHistoryTab(historyTab),
		PendingCount: len(rowsPending),
	}
}

func mapLifeGoalRows(goals []*gen.LifeGoal) []pages.LifeGoalRow {
	views := lifemod.MapGoalViews(goals)
	rows := make([]pages.LifeGoalRow, 0, len(views))
	for _, g := range views {
		rows = append(rows, pages.LifeGoalRow{
			Flag: g.Flag, Title: g.Title, Category: g.Category, Status: g.Status,
			AreaFlag: g.AreaFlag, AreaTitle: g.AreaTitle,
		})
	}
	return rows
}

func mapActiveGoalRows(char *lifemod.CharacterView) []pages.LifeGoalRow {
	all := mapLifeGoalRows(char.Goals)
	goalRows := make([]pages.LifeGoalRow, 0, len(all))
	for _, g := range all {
		if g.Status != pkglife.GoalStatusActive {
			continue
		}
		goalRows = append(goalRows, g)
	}
	return goalRows
}

func mapPendingQuestRows(pending []lifemod.QuestDMView) []pages.LifeQuestRow {
	rows := make([]pages.LifeQuestRow, 0, len(pending))
	for _, item := range pending {
		if item.Quest == nil {
			continue
		}
		q := item.Quest
		row := pages.LifeQuestRow{
			Flag: q.Flag, Title: q.Title, Prompt: q.Prompt, Type: q.Type, Difficulty: q.AiEvaluatedDifficulty,
			Fear: q.AiEvaluatedFear, Exp: q.BaseExpReward, Gold: q.BaseGoldReward, DropTier: q.DropTier,
			DropChance: item.DropChance, Status: q.Status,
			Evidence: mapQuestEvidenceRows(item.Evidence),
		}
		if item.Adjudication != nil {
			adjudication := mapQuestAdjudicationRow(*item.Adjudication)
			row.Adjudication = &adjudication
		}
		rows = append(rows, row)
	}
	return rows
}

func mapCompletedQuestRows(done []*gen.LifeQuest) []pages.LifeQuestRow {
	rows := make([]pages.LifeQuestRow, 0, len(done))
	for _, q := range done {
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
			row.Difficulty = action.Spec.Difficulty
			row.Exp = action.Spec.BaseExpReward
			row.Gold = action.Spec.BaseGoldReward
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

func mapQuestEvidenceRows(items []lifemod.QuestEvidenceView) []pages.LifeQuestEvidenceRow {
	rows := make([]pages.LifeQuestEvidenceRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, pages.LifeQuestEvidenceRow{
			Flag:       item.Flag,
			SourceType: item.SourceType,
			Content:    item.Content,
			SourceURL:  item.SourceURL,
			Summary:    item.Summary,
			When:       item.CreatedAt.UTC().Format("2006-01-02 15:04"),
		})
	}
	return rows
}

func mapQuestAdjudicationRow(item lifemod.QuestAdjudicationView) pages.LifeQuestAdjudicationRow {
	return pages.LifeQuestAdjudicationRow{
		Flag:               item.Flag,
		Status:             item.Status,
		Verdict:            item.Verdict,
		Reason:             item.Reason,
		SuggestedExp:       item.SuggestedExp,
		SuggestedGold:      item.SuggestedGold,
		SuggestedNextSteps: append([]string(nil), item.SuggestedNextSteps...),
		When:               item.CreatedAt.UTC().Format("2006-01-02 15:04"),
	}
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
		return toastErrorKey(ctx, "toast.life.prompt_required")
	}
	goalFlag := strings.TrimSpace(ctx.FormValue("goal_flag"))
	q, _, chance, err := lifeService().CreateQuestFromPrompt(context.Background(), uid, prompt, goalFlag)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	msg := webMsgData(ctx, "toast.life.quest_rated", map[string]any{
		"Difficulty": q.AiEvaluatedDifficulty,
		"Chance":     fmt.Sprintf("%.0f", chance*100),
		"Tier":       q.DropTier,
	})
	setShowToast(ctx, "success", msg)
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeSubmitQuestEvidence(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	content := strings.TrimSpace(ctx.FormValue("content"))
	sourceType := strings.TrimSpace(ctx.FormValue("source_type"))
	sourceURL := strings.TrimSpace(ctx.FormValue("source_url"))
	questFlag := ctx.Params("flag")
	if _, err := lifeService().SubmitQuestEvidence(context.Background(), uid, questFlag, sourceType, content, sourceURL); err != nil {
		flog.Warn("[web] life quest evidence failed uid=%s flag=%s source_type=%s: %v", uid, questFlag, sourceType, err)
		return toastError(ctx, lifeUserError(ctx, err))
	}
	setShowToastKey(ctx, "success", "toast.life.evidence_recorded")
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeAdjudicateQuest(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if _, err := lifeService().AdjudicateQuest(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	setShowToastKey(ctx, "success", "toast.life.dm_suggested_ruling")
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeApplyQuestAdjudication(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().ApplyQuestAdjudication(context.Background(), uid, ctx.Params("flag"), ctx.Params("adjudicationFlag")); err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	setShowToastKey(ctx, "success", "toast.life.ruling_applied")
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	msg := fmt.Sprintf("+%d EXP, +%d gold (dice %.2f)", res.GainedExp, res.GainedGold, res.Dice)
	if res.Dropped {
		msg = fmt.Sprintf("Loot drop! %s — +%d EXP, +%d gold", res.ItemName, res.GainedExp, res.GainedGold)
	}
	if len(res.NewlyUnlocked) > 0 {
		names := make([]string, 0, len(res.NewlyUnlocked))
		for _, a := range res.NewlyUnlocked {
			names = append(names, a.Name)
		}
		msg = fmt.Sprintf("%s · Achievement: %s", msg, strings.Join(names, ", "))
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
		return toastError(ctx, lifeUserError(ctx, err))
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
		return toastError(ctx, lifeUserError(ctx, err))
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
		return toastError(ctx, lifeUserError(ctx, err))
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	setShowToastKey(ctx, "warning", "toast.life.quest_failed_tarnish")
	ctx.Set("HX-Redirect", "/service/web/life/quests")
	return ctx.SendStatus(http.StatusOK)
}

func lifeDismissQuest(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	if err := lifeService().DismissQuest(context.Background(), uid, ctx.Params("flag")); err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	setShowToastKey(ctx, "success", "toast.life.quest_dismissed")
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
	backpackPage := parsePositiveIntQuery(ctx, "backpack_page", 1)
	page, err := lifeService().ListInventoryPage(context.Background(), uid, backpackPage, pages.LifeDefaultListPerPage)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	equippedByID := lifeEquippedSlotFields(page.Slots)
	backpackRows := mapLifeInventoryRows(page.Items, equippedByID)
	equipRows := mapLifeInventoryRows(page.Equipped, equippedByID)
	pending, err := lifeService().ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Type("html")
	return pages.LifeInventoryPage(pages.LifeInventoryData{
		Slots:        pages.LifeBuildEquipSlots(equipRows),
		Items:        backpackRows,
		BackpackPage: pages.LifeWithBackpackPager(pages.LifeBuildPageInfo(backpackPage, pages.LifeDefaultListPerPage, page.Total)),
		PendingCount: len(pending),
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func lifeEquippedSlotFields(slots *gen.LifeEquippedSlots) map[int64]string {
	equipped := map[int64]string{}
	if slots == nil {
		return equipped
	}
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
	return equipped
}

func mapLifeInventoryRows(items []lifemod.InventoryItem, equipped map[int64]string) []pages.LifeInventoryRow {
	rows := make([]pages.LifeInventoryRow, 0, len(items))
	for _, it := range items {
		if it.Inv == nil || it.Equip == nil {
			continue
		}
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
			SlotField:  slotField,
			BuffText:   pages.LifeFormatBuffText(it.Equip.StatBuffs),
			PerkText:   pages.LifeFormatPerkText(it.Equip.AiUnlockedPrivilege),
			Lore:       lore,
			LoreStatus: it.Inv.LoreStatus,
			Equipped:   slotField != "", Tarnished: pkglife.IsTarnished(it.Inv.TarnishedUntil, time.Now()),
		})
	}
	return rows
}

func lifeAchievementsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	uid, err := lifeUID(ctx)
	if err != nil {
		return err
	}
	items, err := lifeService().ListAchievements(context.Background(), uid)
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	pending, err := lifeService().ListQuests(context.Background(), uid, "Pending")
	if err != nil {
		return toastError(ctx, lifeUserError(ctx, err))
	}
	rows := make([]pages.LifeAchievementRow, 0, len(items))
	unlockedCount := 0
	for _, it := range items {
		if it.Unlocked {
			unlockedCount++
		}
		unlockedAt := ""
		if it.UnlockedAt != nil {
			unlockedAt = it.UnlockedAt.Format("2006-01-02")
		}
		rows = append(rows, pages.LifeAchievementRow{
			Flag: it.Flag, Name: it.Name, Description: it.Description,
			Unlocked: it.Unlocked, UnlockedAt: unlockedAt,
			ShowProgress: it.ShowProgress, Current: it.Current, Target: it.Target, Retired: it.Retired,
		})
	}
	ctx.Type("html")
	return pages.LifeAchievementsPage(pages.LifeAchievementsData{
		Items: rows, UnlockedCount: unlockedCount, PendingCount: len(pending),
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
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
		return toastError(ctx, lifeUserError(ctx, err))
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
		return toastError(ctx, lifeUserError(ctx, err))
	}
	ctx.Set("HX-Redirect", "/service/web/life/inventory")
	return ctx.SendStatus(http.StatusOK)
}
