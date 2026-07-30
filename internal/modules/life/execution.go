package life

import (
	"context"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	caplife "github.com/flowline-io/flowbot/pkg/capability"
	lifecap "github.com/flowline-io/flowbot/pkg/capability/life"
	"github.com/flowline-io/flowbot/pkg/hub"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
	"github.com/flowline-io/flowbot/pkg/types"
)

// TodayActionView is one pending action occurrence with definition metadata.
type TodayActionView struct {
	Occurrence *gen.LifeActionOccurrence
	Node       *gen.LifePlanNode
	Spec       *gen.LifeActionSpec
}

// TodayHabitView is one habit plus today's check-in state.
type TodayHabitView struct {
	Node      *gen.LifePlanNode
	Spec      *gen.LifeActionSpec
	Checkin   *gen.LifeHabitCheckin
	CheckedIn bool
}

// TodayBoardView is the execution board rendered on the quests page.
type TodayBoardView struct {
	Actions []TodayActionView
	Habits  []TodayHabitView
}

// ListTodayActions syncs recurring actions and returns today's action/habit board.
func (s *Service) ListTodayActions(ctx context.Context, userID string) (*TodayBoardView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.store.EnsureRecurringOccurrences(ctx, p.ID, now); err != nil {
		return nil, err
	}
	nodes, err := s.store.ListPlanNodes(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	specs, err := s.store.ListActionSpecs(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	occurrences, err := s.store.ListActionOccurrences(ctx, p.ID, "pending")
	if err != nil {
		return nil, err
	}
	day := startOfDay(now)
	nodeByID := mapPlanNodesByID(nodes)
	specByNodeID := mapActionSpecsByNodeID(specs)
	actions := buildTodayActionViews(occurrences, nodeByID, specByNodeID)
	habits, err := s.buildTodayHabitViews(ctx, p.ID, specs, nodeByID, day)
	if err != nil {
		return nil, err
	}
	return &TodayBoardView{Actions: actions, Habits: habits}, nil
}

// CompleteActionOccurrence marks an execution occurrence completed and grants rewards.
func (s *Service) CompleteActionOccurrence(ctx context.Context, userID, occurrenceFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	occurrence, err := s.store.GetActionOccurrenceByFlag(ctx, p.ID, occurrenceFlag)
	if err != nil {
		return err
	}
	if occurrence == nil {
		return lifeNotFound("occurrence not found")
	}
	node, err := s.store.GetPlanNode(ctx, occurrence.PlanNodeID)
	if err != nil {
		return err
	}
	if node == nil {
		return lifeNotFound("action not found")
	}
	spec, err := s.store.GetActionSpecByPlanNodeID(ctx, node.ID)
	if err != nil {
		return err
	}
	baseExp, baseGold := defaultActionRewards(spec)
	slots, err := s.store.GetEquippedSlots(ctx, p.ID)
	if err != nil {
		return err
	}
	buffs, err := s.equippedBuffs(ctx, slots)
	if err != nil {
		return err
	}
	goldReward := max(int(float64(baseGold)*buffs.GoldMult), 0)
	skill, char, err := s.resolveDefaultActionSkill(ctx, p.ID)
	if err != nil {
		return err
	}
	casc := pkglife.ApplyCascade(pkglife.CascadeInput{
		BaseExp:                  baseExp,
		BaseGold:                 goldReward,
		Skill:                    pkglife.StatSnapshot{Level: skill.Level, CurrentExp: skill.CurrentExp},
		Characteristic:           pkglife.StatSnapshot{Level: char.Level, CurrentExp: char.CurrentExp},
		Profile:                  pkglife.StatSnapshot{Level: p.Level, CurrentExp: p.Exp},
		ProfileGold:              p.Gold,
		ExpToCharacteristicRatio: skill.ExpToCharacteristicRatio,
	})
	return s.store.CompleteActionOccurrence(ctx, store.LifeCompleteOccurrenceInput{
		OccurrenceID: occurrence.ID,
		ProfileID:    p.ID,
		PlanNodeID:   node.ID,
		Summary:      node.Title,
		GainedExp:    casc.GainedExp,
		GainedGold:   casc.GainedGold,
		SkillID:      skill.ID,
		CharID:       char.ID,
		SkillLevel:   casc.Skill.Level,
		SkillExp:     casc.Skill.CurrentExp,
		CharLevel:    casc.Characteristic.Level,
		CharExp:      casc.Characteristic.CurrentExp,
		ProfLevel:    casc.Profile.Level,
		ProfExp:      casc.Profile.CurrentExp,
		ProfGold:     casc.ProfileGold,
	})
}

func defaultActionRewards(spec *gen.LifeActionSpec) (baseExp, baseGold int) {
	if spec == nil {
		_, baseExp, baseGold, _ = pkglife.DefaultRewards("C")
		return baseExp, baseGold
	}
	return spec.BaseExpReward, spec.BaseGoldReward
}

func (s *Service) resolveDefaultActionSkill(ctx context.Context, profileID int64) (*gen.LifeSkill, *gen.LifeCharacteristic, error) {
	skill, err := s.resolveSkillForEvaluation(ctx, profileID, &lifecap.QuestEvaluation{
		SkillName: "Exploration",
		StatCode:  "FOC",
	})
	if err != nil {
		return nil, nil, err
	}
	char, err := s.store.GetCharacteristic(ctx, skill.CharacteristicID)
	if err != nil || char == nil {
		return nil, nil, lifeNotFound("characteristic missing")
	}
	return skill, char, nil
}

// SkipActionOccurrence marks a pending occurrence skipped.
func (s *Service) SkipActionOccurrence(ctx context.Context, userID, occurrenceFlag string) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	occurrence, err := s.store.GetActionOccurrenceByFlag(ctx, p.ID, occurrenceFlag)
	if err != nil {
		return err
	}
	if occurrence == nil {
		return lifeNotFound("occurrence not found")
	}
	return s.store.SkipActionOccurrence(ctx, store.LifeSkipOccurrenceInput{
		OccurrenceID: occurrence.ID,
		State:        "skipped",
	})
}

// CheckInHabit writes one daily habit check-in.
func (s *Service) CheckInHabit(ctx context.Context, userID, nodeFlag string, at time.Time) error {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return err
	}
	node, err := s.store.GetPlanNodeByFlag(ctx, p.ID, nodeFlag)
	if err != nil {
		return err
	}
	if node == nil {
		return lifeNotFound("action not found")
	}
	spec, err := s.store.GetActionSpecByPlanNodeID(ctx, node.ID)
	if err != nil {
		return err
	}
	if spec == nil || spec.TaskType != "habit" {
		return lifeConflict("habit check-in not available")
	}
	_, err = s.store.UpsertHabitCheckin(ctx, store.LifeHabitCheckinInput{
		ProfileID:  p.ID,
		PlanNodeID: node.ID,
		CheckinAt:  at,
		Status:     "done",
		Summary:    node.Title,
	})
	return err
}

// ListHabitCheckins returns check-ins in a date range.
func (s *Service) ListHabitCheckins(ctx context.Context, userID, nodeFlag string, from, to time.Time) ([]*gen.LifeHabitCheckin, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	node, err := s.store.GetPlanNodeByFlag(ctx, p.ID, nodeFlag)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, lifeNotFound("action not found")
	}
	return s.store.ListHabitCheckins(ctx, p.ID, node.ID, from, to)
}

// PreviewGoalBreakdown asks the life capability for a suggested tree.
func (s *Service) PreviewGoalBreakdown(ctx context.Context, userID, rootTitle, description string) (*lifecap.GoalBreakdownSuggestion, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	if err := s.store.EnsureAIContext(ctx, p.ID); err != nil {
		return nil, err
	}
	aiCtx, err := s.store.GetAIContext(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	goals, err := s.store.ListGoals(ctx, p.ID, "Active")
	if err != nil {
		return nil, err
	}
	activeGoals := make([]string, 0, len(goals))
	for _, goal := range goals {
		activeGoals = append(activeGoals, goal.Title)
	}
	slots, err := s.store.GetEquippedSlots(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	privs, err := s.equippedPrivileges(ctx, slots)
	if err != nil {
		return nil, err
	}
	params := map[string]any{
		"root_title":      strings.TrimSpace(rootTitle),
		"description":     strings.TrimSpace(description),
		"active_goals":    activeGoals,
		"privileges":      privs,
		"breakdown_depth": privilegeDepth(privs),
	}
	if aiCtx != nil {
		params["ai_personality"] = aiCtx.AiDmPersonality
	}
	res, err := caplife.Invoke(ctx, hub.CapLife, lifecap.OpBreakdownGoalTree, params)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "life: breakdown goal tree", err)
	}
	return decodeGoalBreakdown(res.Data)
}

// ImportGoalBreakdown persists a suggestion tree as plan nodes.
func (s *Service) ImportGoalBreakdown(ctx context.Context, userID string, suggestion *lifecap.GoalBreakdownSuggestion) error {
	if suggestion == nil {
		return lifeInvalid("breakdown suggestion required")
	}
	suggestion = normalizeBreakdownForImport(suggestion, "")
	if suggestion == nil {
		return lifeInvalid("breakdown suggestion required")
	}
	return s.store.WithTx(ctx, func(txStore *store.LifeStore) error {
		txService := *s
		txService.store = txStore
		_, err := txService.importGoalBreakdownNode(ctx, userID, "", suggestion)
		return err
	})
}

func (s *Service) importGoalBreakdownNode(ctx context.Context, userID, parentFlag string, suggestion *lifecap.GoalBreakdownSuggestion) (*PlanNodeView, error) {
	var action *ActionInput
	if suggestion.Action != nil {
		action = &ActionInput{
			IsRepeatable:       suggestion.Action.IsRepeatable,
			TrackingMode:       suggestion.Action.TrackingMode,
			RepeatTrigger:      suggestion.Action.RepeatTrigger,
			SuggestedCadence:   suggestion.Action.SuggestedCadence,
			IsIdentityBuilding: suggestion.Action.IsIdentityBuilding,
			Reason:             suggestion.Action.Reason,
			Difficulty:         suggestion.Action.Difficulty,
			BaseExp:            suggestion.Action.BaseExp,
			BaseGold:           suggestion.Action.BaseGold,
		}
	}
	row, err := s.CreatePlanNode(ctx, userID, parentFlag, suggestion.NodeType, suggestion.Title, suggestion.Description, action)
	if err != nil {
		return nil, err
	}
	for _, child := range suggestion.Children {
		if _, err := s.importGoalBreakdownNode(ctx, userID, row.Node.Flag, &child); err != nil {
			return nil, err
		}
	}
	return row, nil
}

func decodeGoalBreakdown(data any) (*lifecap.GoalBreakdownSuggestion, error) {
	if tree, ok := data.(*lifecap.GoalBreakdownSuggestion); ok {
		return tree, nil
	}
	raw, err := sonic.Marshal(data)
	if err != nil {
		return nil, lifeInvalid("unexpected breakdown payload")
	}
	tree := &lifecap.GoalBreakdownSuggestion{}
	if err := sonic.Unmarshal(raw, tree); err != nil {
		return nil, types.WrapError(types.ErrInvalidArgument, "life: decode breakdown payload", err)
	}
	return tree, nil
}

func startOfDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func normalizeBreakdownForImport(node *lifecap.GoalBreakdownSuggestion, parentType string) *lifecap.GoalBreakdownSuggestion {
	if node == nil {
		return nil
	}
	title := strings.TrimSpace(node.Title)
	description := strings.TrimSpace(node.Description)
	if title == "" {
		title = fallbackBreakdownTitle(description)
	}
	if title == "" {
		return nil
	}
	normalized := &lifecap.GoalBreakdownSuggestion{
		NodeType:    normalizedImportNodeType(parentType, strings.TrimSpace(node.NodeType), len(node.Children) > 0, node.Action != nil),
		Title:       title,
		Description: description,
	}
	if normalized.NodeType == "action" {
		normalized.Action = node.Action
		return normalized
	}
	children := make([]lifecap.GoalBreakdownSuggestion, 0, len(node.Children))
	for _, child := range node.Children {
		childCopy := normalizeBreakdownForImport(&child, normalized.NodeType)
		if childCopy == nil {
			continue
		}
		children = append(children, *childCopy)
	}
	normalized.Children = children
	return normalized
}

func normalizedImportNodeType(parentType, requested string, hasChildren, hasAction bool) string {
	switch parentType {
	case "":
		return "goal"
	case "goal":
		if strings.EqualFold(strings.TrimSpace(requested), "project") || strings.EqualFold(strings.TrimSpace(requested), "action") || hasAction {
			return "project"
		}
		if hasChildren {
			return "milestone"
		}
		return "project"
	case "milestone":
		return "project"
	case "project":
		return "action"
	default:
		return "action"
	}
}

func fallbackBreakdownTitle(description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return ""
	}
	runes := []rune(description)
	if len(runes) > 32 {
		runes = runes[:32]
	}
	return strings.TrimSpace(string(runes))
}

func mapPlanNodesByID(nodes []*gen.LifePlanNode) map[int64]*gen.LifePlanNode {
	out := make(map[int64]*gen.LifePlanNode, len(nodes))
	for _, node := range nodes {
		out[node.ID] = node
	}
	return out
}

func mapActionSpecsByNodeID(specs []*gen.LifeActionSpec) map[int64]*gen.LifeActionSpec {
	out := make(map[int64]*gen.LifeActionSpec, len(specs))
	for _, spec := range specs {
		out[spec.PlanNodeID] = spec
	}
	return out
}

func buildTodayActionViews(occurrences []*gen.LifeActionOccurrence, nodeByID map[int64]*gen.LifePlanNode, specByNodeID map[int64]*gen.LifeActionSpec) []TodayActionView {
	out := make([]TodayActionView, 0, len(occurrences))
	for _, occurrence := range occurrences {
		node := nodeByID[occurrence.PlanNodeID]
		if node == nil || node.Status != "Active" {
			continue
		}
		out = append(out, TodayActionView{
			Occurrence: occurrence,
			Node:       node,
			Spec:       specByNodeID[occurrence.PlanNodeID],
		})
	}
	return out
}

func (s *Service) buildTodayHabitViews(ctx context.Context, profileID int64, specs []*gen.LifeActionSpec, nodeByID map[int64]*gen.LifePlanNode, day time.Time) ([]TodayHabitView, error) {
	out := make([]TodayHabitView, 0)
	end := day.Add(24*time.Hour - time.Nanosecond)
	for _, spec := range specs {
		if spec.TaskType != "habit" {
			continue
		}
		node := nodeByID[spec.PlanNodeID]
		if node == nil || node.Status != "Active" {
			continue
		}
		checkins, err := s.store.ListHabitCheckins(ctx, profileID, node.ID, day, end)
		if err != nil {
			return nil, err
		}
		view := TodayHabitView{Node: node, Spec: spec}
		if len(checkins) > 0 {
			view.Checkin = checkins[len(checkins)-1]
			view.CheckedIn = true
		}
		out = append(out, view)
	}
	return out, nil
}
