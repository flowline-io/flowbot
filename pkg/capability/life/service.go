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
}
