package types

import "strings"

// Token usage source constants classify LLM consumption by execution context.
const (
	TokenUsageSourceAgent         = "agent"
	TokenUsageSourcePipeline      = "pipeline"
	TokenUsageSourceScheduledTask = "scheduled_task"
	TokenUsageSourceSubagent      = "subagent"
)

// TokenUsageSourceOrder is the stable display order for usage-type charts.
var TokenUsageSourceOrder = []string{
	TokenUsageSourceAgent,
	TokenUsageSourcePipeline,
	TokenUsageSourceScheduledTask,
	TokenUsageSourceSubagent,
}

// NormalizeTokenUsageSource maps persisted source values to a canonical usage type.
func NormalizeTokenUsageSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case TokenUsageSourceAgent, "chat_agent", "interactive":
		return TokenUsageSourceAgent
	case TokenUsageSourcePipeline:
		return TokenUsageSourcePipeline
	case TokenUsageSourceScheduledTask, "scheduled":
		return TokenUsageSourceScheduledTask
	case TokenUsageSourceSubagent:
		return TokenUsageSourceSubagent
	default:
		if strings.TrimSpace(source) == "" {
			return TokenUsageSourceAgent
		}
		return TokenUsageSourceAgent
	}
}

// TokenUsageSourceLabel returns a human-readable label for charts and legends.
func TokenUsageSourceLabel(source string) string {
	switch NormalizeTokenUsageSource(source) {
	case TokenUsageSourcePipeline:
		return "Pipeline"
	case TokenUsageSourceScheduledTask:
		return "Scheduled Task"
	case TokenUsageSourceSubagent:
		return "Subagent"
	default:
		return "Agent"
	}
}
