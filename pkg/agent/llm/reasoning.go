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
	_, ok := thinkingRequestProfileFor(modelName)
	return ok
}

// ReasoningCallOptions returns per-request call options that enable provider reasoning streams.
// Langchaingo applies extended thinking through GenerateContent options rather than model construction.
func ReasoningCallOptions(modelName string, maxTokens int, thinkingLevel string) []llms.CallOption {
	level := NormalizeThinkingLevel(thinkingLevel)
	if level == ThinkingLevelOff {
		return nil
	}
	if !SupportsReasoningStream(modelName) {
		return nil
	}

	opts := []llms.CallOption{
		llms.WithReturnThinking(true),
		llms.WithStreamThinking(true),
	}

	if isAnthropicReasoningModel(modelName) {
		mode := anthropicThinkingMode(level)
		if mode == llms.ThinkingModeNone {
			return nil
		}
		opts = append(opts, llms.WithThinkingMode(mode))
		if maxTokens > 0 && mode != llms.ThinkingModeAuto {
			budget := llms.CalculateThinkingBudget(mode, maxTokens)
			if budget > 0 {
				opts = append(opts, llms.WithThinkingBudget(budget))
			}
		}
		if mode == llms.ThinkingModeAuto && maxTokens > 0 {
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

// isMiMoReasoningModel reports whether model is a Xiaomi MiMo V2.5 series
// reasoning model that returns reasoning_content (see MiMo deep-thinking API).
func isMiMoReasoningModel(modelName string) bool {
	lower := strings.ToLower(modelName)
	return strings.Contains(lower, "mimo-v2.5")
}

// needsThinkingHTTPClient reports whether the OpenAI-compatible HTTP client
// should inject provider thinking request fields (thinking.type, etc.).
func needsThinkingHTTPClient(modelName string) bool {
	_, ok := thinkingRequestProfileFor(modelName)
	return ok
}

func isAnthropicReasoningModel(modelName string) bool {
	if resolveModel(modelName).Provider == ProviderAnthropic {
		return true
	}
	return strings.Contains(strings.ToLower(modelName), "claude")
}
