// Package life is the Life capability (quest evaluation and item lore).
package life

import "context"

// QuestEvaluation is the structured LLM assessment of a quest prompt.
type QuestEvaluation struct {
	Title      string `json:"title"`
	SkillName  string `json:"skill_name"`
	StatCode   string `json:"stat_code"`
	Difficulty string `json:"difficulty"`
	Fear       int    `json:"fear"`
	BaseExp    int    `json:"base_exp"`
	BaseGold   int    `json:"base_gold"`
	DropTier   string `json:"drop_tier"`
	QuestType  string `json:"quest_type"`
}

// InstanceLore is generated flavor text for a dropped item instance.
type InstanceLore struct {
	Name string `json:"name"`
	Lore string `json:"lore"`
}

// GoalBreakdownRequest carries the root goal plus planning context.
type GoalBreakdownRequest struct {
	RootTitle      string         `json:"root_title"`
	Description    string         `json:"description,omitempty"`
	AIPersonality  string         `json:"ai_personality,omitempty"`
	Privileges     map[string]any `json:"privileges,omitempty"`
	ActiveGoals    []string       `json:"active_goals,omitempty"`
	BreakdownDepth string         `json:"breakdown_depth,omitempty"`
}

// GoalBreakdownActionSuggestion is the suggested action payload for action nodes.
type GoalBreakdownActionSuggestion struct {
	IsRepeatable       bool   `json:"is_repeatable,omitempty"`
	TrackingMode       string `json:"tracking_mode,omitempty"`
	RepeatTrigger      string `json:"repeat_trigger,omitempty"`
	SuggestedCadence   string `json:"suggested_cadence,omitempty"`
	IsIdentityBuilding bool   `json:"is_identity_building,omitempty"`
	Reason             string `json:"reason,omitempty"`
}

// GoalBreakdownSuggestion is one suggested tree node.
type GoalBreakdownSuggestion struct {
	NodeType    string                         `json:"node_type"`
	Title       string                         `json:"title"`
	Description string                         `json:"description,omitempty"`
	Action      *GoalBreakdownActionSuggestion `json:"action,omitempty"`
	Children    []GoalBreakdownSuggestion      `json:"children,omitempty"`
}

// EvaluateQuestRequest carries prompt plus optional DM context and gear privileges.
type EvaluateQuestRequest struct {
	Prompt         string         `json:"prompt"`
	AIPersonality  string         `json:"ai_personality,omitempty"`
	CompletionRate float64        `json:"completion_rate,omitempty"`
	Mood           map[string]any `json:"mood,omitempty"`
	Privileges     map[string]any `json:"privileges,omitempty"`
	ActiveGoals    []string       `json:"active_goals,omitempty"`
	BreakdownDepth string         `json:"breakdown_depth,omitempty"`
}

// Service is the life capability contract.
type Service interface {
	EvaluateQuest(ctx context.Context, req EvaluateQuestRequest) (*QuestEvaluation, error)
	GenerateInstanceLore(ctx context.Context, questTitle, equipmentName, rarity string) (*InstanceLore, error)
	BreakdownGoalTree(ctx context.Context, req GoalBreakdownRequest) (*GoalBreakdownSuggestion, error)
}
