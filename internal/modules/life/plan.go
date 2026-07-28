package life

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	pkglife "github.com/flowline-io/flowbot/pkg/life"
)

// ActionInput captures the structured fields used to classify an action node.
type ActionInput struct {
	TaskType           string
	IsRepeatable       bool
	TrackingMode       string
	RepeatTrigger      string
	SuggestedCadence   string
	IsIdentityBuilding bool
	Reason             string
	Difficulty         string
	BaseExp            int
	BaseGold           int
	DependencyFlags    []string
}

// PlanNodeView is one rendered tree node with optional action metadata.
type PlanNodeView struct {
	Node     *gen.LifePlanNode
	Action   *gen.LifeActionSpec
	Children []*PlanNodeView
}

var allowedPlanChildren = map[string][]string{
	"goal":      {"milestone", "project"},
	"milestone": {"project"},
	"project":   {"action"},
	"action":    {},
}

// CreatePlanNode creates one plan node under an optional parent.
func (s *Service) CreatePlanNode(ctx context.Context, userID, parentFlag, nodeType, title, description string, action *ActionInput) (*PlanNodeView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
		return nil, err
	}
	planInput, err := s.preparePlanNodeCreate(ctx, p.ID, parentFlag, nodeType, title, description, action)
	if err != nil {
		return nil, err
	}
	row, spec, err := s.store.CreatePlanNode(ctx, store.LifeCreatePlanNodeInput{
		ProfileID:             p.ID,
		ParentID:              planInput.ParentID,
		NodeType:              planInput.NodeType,
		Title:                 planInput.Title,
		Description:           planInput.Description,
		Status:                "Active",
		ActionSpec:            planInput.ActionSpec,
		DependencyPlanNodeIDs: planInput.DependencyPlanNodeIDs,
	})
	if err != nil {
		return nil, err
	}
	if spec != nil && spec.TaskType == "todo" {
		if _, err := s.store.EnsureTodoOccurrence(ctx, p.ID, row.ID); err != nil {
			return nil, err
		}
	}
	return &PlanNodeView{Node: row, Action: spec}, nil
}

type preparedPlanNodeCreate struct {
	ParentID              *int64
	ActionSpec            *store.LifePlanActionSpecInput
	NodeType              string
	Title                 string
	Description           string
	DependencyPlanNodeIDs []int64
}

func (s *Service) preparePlanNodeCreate(ctx context.Context, profileID int64, parentFlag, nodeType, title, description string, action *ActionInput) (*preparedPlanNodeCreate, error) {
	normalizedType := normalizePlanNodeType(nodeType)
	if normalizedType == "" {
		return nil, lifeInvalid("invalid node type")
	}
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		return nil, lifeInvalid("title required")
	}
	parentID, err := s.resolvePlanNodeParent(ctx, profileID, parentFlag, normalizedType)
	if err != nil {
		return nil, err
	}
	actionSpec, err := buildPlanActionSpec(normalizedType, action)
	if err != nil {
		return nil, err
	}
	dependencyIDs, err := s.resolveActionDependencies(ctx, profileID, parentID, actionSpec, action)
	if err != nil {
		return nil, err
	}
	return &preparedPlanNodeCreate{
		ParentID:              parentID,
		ActionSpec:            actionSpec,
		NodeType:              normalizedType,
		Title:                 normalizedTitle,
		Description:           strings.TrimSpace(description),
		DependencyPlanNodeIDs: dependencyIDs,
	}, nil
}

func (s *Service) resolvePlanNodeParent(ctx context.Context, profileID int64, parentFlag, nodeType string) (*int64, error) {
	if parentFlag == "" {
		if nodeType != "goal" {
			return nil, lifeInvalid("non-goal node requires parent")
		}
		return nil, nil
	}
	parent, err := s.store.GetPlanNodeByFlag(ctx, profileID, parentFlag)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, lifeNotFound("parent not found")
	}
	if !isAllowedPlanChild(parent.NodeType, nodeType) {
		return nil, lifeInvalid("invalid parent child relation")
	}
	return &parent.ID, nil
}

func buildPlanActionSpec(nodeType string, action *ActionInput) (*store.LifePlanActionSpecInput, error) {
	if nodeType != "action" {
		return nil, nil
	}
	if action == nil {
		return nil, lifeInvalid("action details required")
	}
	return classifyActionInput(action), nil
}

func (s *Service) resolveActionDependencies(ctx context.Context, profileID int64, parentID *int64, spec *store.LifePlanActionSpecInput, action *ActionInput) ([]int64, error) {
	if spec == nil || spec.TaskType != "checkpoint" {
		return nil, nil
	}
	if parentID == nil {
		return nil, lifeInvalid("checkpoint action requires parent")
	}
	flags := normalizeDependencyFlags(action.DependencyFlags)
	if len(flags) == 0 {
		return nil, lifeInvalid("checkpoint dependencies required")
	}
	ids := make([]int64, 0, len(flags))
	for _, flag := range flags {
		node, err := s.store.GetPlanNodeByFlag(ctx, profileID, flag)
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, lifeNotFound("dependency action not found")
		}
		ids = append(ids, node.ID)
	}
	return ids, nil
}

// ConfirmHabitAction marks a habit candidate as confirmed.
func (s *Service) ConfirmHabitAction(ctx context.Context, userID, nodeFlag string) error {
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
	if spec == nil || spec.TaskType != "habit_candidate" {
		return lifeConflict("habit confirmation not required")
	}
	_, err = s.store.ConfirmHabitAction(ctx, node.ID)
	return err
}

// ListPlanTree returns the profile planning tree.
func (s *Service) ListPlanTree(ctx context.Context, userID string) ([]*PlanNodeView, error) {
	p, err := s.EnsureProfile(ctx, userID, "", config.DefaultClass)
	if err != nil {
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
	return buildPlanTree(nodes, specs), nil
}

func normalizePlanNodeType(nodeType string) string {
	switch strings.ToLower(strings.TrimSpace(nodeType)) {
	case "goal":
		return "goal"
	case "milestone":
		return "milestone"
	case "project":
		return "project"
	case "action":
		return "action"
	default:
		return ""
	}
}

func normalizeTrackingMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "consistency":
		return "consistency"
	default:
		return "completion"
	}
}

func normalizeRepeatTrigger(trigger string) string {
	switch strings.ToLower(strings.TrimSpace(trigger)) {
	case "time":
		return "time"
	case "condition":
		return "condition"
	default:
		return "none"
	}
}

func isAllowedPlanChild(parentType, nodeType string) bool {
	children := allowedPlanChildren[parentType]
	return slices.Contains(children, nodeType)
}

func classifyActionInput(in *ActionInput) *store.LifePlanActionSpecInput {
	difficulty, baseExp, baseGold := actionRewardFields(in)
	if strings.EqualFold(strings.TrimSpace(in.TaskType), "checkpoint") {
		return &store.LifePlanActionSpecInput{
			TaskType:           "checkpoint",
			TrackingMode:       "completion",
			RepeatTrigger:      "condition",
			IsIdentityBuilding: in.IsIdentityBuilding,
			Reason:             strings.TrimSpace(in.Reason),
			Difficulty:         difficulty,
			BaseExpReward:      baseExp,
			BaseGoldReward:     baseGold,
		}
	}
	taskType := "todo"
	trackingMode := normalizeTrackingMode(in.TrackingMode)
	needsConfirmation := false
	if in.IsRepeatable {
		if trackingMode == "consistency" {
			taskType = "habit_candidate"
			needsConfirmation = true
		} else {
			taskType = "recurring"
		}
	}
	return &store.LifePlanActionSpecInput{
		TaskType:              taskType,
		TrackingMode:          trackingMode,
		IsRepeatable:          in.IsRepeatable,
		RepeatTrigger:         normalizeRepeatTrigger(in.RepeatTrigger),
		SuggestedCadence:      strings.TrimSpace(in.SuggestedCadence),
		IsIdentityBuilding:    in.IsIdentityBuilding,
		Reason:                strings.TrimSpace(in.Reason),
		NeedsUserConfirmation: needsConfirmation,
		Difficulty:            difficulty,
		BaseExpReward:         baseExp,
		BaseGoldReward:        baseGold,
	}
}

func actionRewardFields(in *ActionInput) (difficulty string, baseExp int, baseGold int) {
	if in == nil {
		_, baseExp, baseGold, _ = pkglife.DefaultRewards("C")
		return "C", baseExp, baseGold
	}
	difficulty = pkglife.NormalizeDifficulty(in.Difficulty)
	_, baseExp, baseGold, _ = pkglife.DefaultRewards(difficulty)
	return difficulty, baseExp, baseGold
}

func normalizeDependencyFlags(flags []string) []string {
	if len(flags) == 0 {
		return nil
	}
	out := make([]string, 0, len(flags))
	seen := make(map[string]struct{}, len(flags))
	for _, flag := range flags {
		normalized := strings.TrimSpace(flag)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func buildPlanTree(nodes []*gen.LifePlanNode, specs []*gen.LifeActionSpec) []*PlanNodeView {
	if len(nodes) == 0 {
		return nil
	}
	byID := make(map[int64]*PlanNodeView, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = &PlanNodeView{Node: node}
	}
	for _, spec := range specs {
		if view, ok := byID[spec.PlanNodeID]; ok {
			view.Action = spec
		}
	}
	roots := make([]*PlanNodeView, 0)
	for _, node := range nodes {
		view := byID[node.ID]
		if node.ParentID == nil {
			roots = append(roots, view)
			continue
		}
		parent, ok := byID[*node.ParentID]
		if !ok {
			roots = append(roots, view)
			continue
		}
		parent.Children = append(parent.Children, view)
	}
	sortPlanChildren(roots)
	return roots
}

func sortPlanChildren(nodes []*PlanNodeView) {
	slices.SortFunc(nodes, func(a, b *PlanNodeView) int {
		if a.Node.SortOrder != b.Node.SortOrder {
			if a.Node.SortOrder < b.Node.SortOrder {
				return -1
			}
			return 1
		}
		if a.Node.CreatedAt.Before(b.Node.CreatedAt) {
			return -1
		}
		if a.Node.CreatedAt.After(b.Node.CreatedAt) {
			return 1
		}
		return 0
	})
	for _, node := range nodes {
		if len(node.Children) > 0 {
			sortPlanChildren(node.Children)
		}
	}
}

func confirmedAtValue(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
