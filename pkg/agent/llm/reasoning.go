package llm

import (
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// SupportsReasoningStream reports whether a model should use reasoning stream callbacks.
func SupportsReasoningStream(modelName string) bool {
	if llms.IsReasoningModel(modelName) {
		return true
	}
	return isDeepSeekV4ReasoningModel(modelName)
}

// ReasoningCallOptions returns per-request call options that enable provider reasoning streams.
// Langchaingo applies extended thinking through GenerateContent options rather than model construction.
func ReasoningCallOptions(modelName string, maxTokens int) []llms.CallOption {
	if !SupportsReasoningStream(modelName) {
		return nil
	}

	opts := []llms.CallOption{
		llms.WithReturnThinking(true),
		llms.WithStreamThinking(true),
	}

	if isAnthropicReasoningModel(modelName) {
		opts = append(opts, llms.WithThinkingMode(llms.ThinkingModeAuto))
		if maxTokens > 0 {
			budget := llms.CalculateThinkingBudget(llms.ThinkingModeMedium, maxTokens)
			if budget > 0 {
				opts = append(opts, llms.WithThinkingBudget(budget))
			}
		}
	}

	return opts
}

func isDeepSeekV4ReasoningModel(modelName string) bool {
	lower := strings.ToLower(modelName)
	return strings.Contains(lower, "deepseek-v4")
}

func isAnthropicReasoningModel(modelName string) bool {
	if resolveModel(modelName).Provider == ProviderAnthropic {
		return true
	}
	return strings.Contains(strings.ToLower(modelName), "claude")
}
