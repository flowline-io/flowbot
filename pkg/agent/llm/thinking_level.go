package llm

import (
	"strings"

	"github.com/tmc/langchaingo/llms"
)

const (
	// ThinkingLevelDefault preserves provider-specific server defaults.
	ThinkingLevelDefault = "default"
	// ThinkingLevelOff disables extended reasoning when supported.
	ThinkingLevelOff = "off"
	// ThinkingLevelLow requests a light reasoning budget.
	ThinkingLevelLow = "low"
	// ThinkingLevelMedium requests a moderate reasoning budget.
	ThinkingLevelMedium = "medium"
	// ThinkingLevelHigh requests a high reasoning budget.
	ThinkingLevelHigh = "high"
)

// ValidThinkingLevel reports whether level is a supported UI value.
func ValidThinkingLevel(level string) bool {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", ThinkingLevelDefault, ThinkingLevelOff, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh:
		return true
	default:
		return false
	}
}

// NormalizeThinkingLevel trims and lowercases a thinking level, defaulting empty to default.
func NormalizeThinkingLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		return ThinkingLevelDefault
	}
	return level
}

// anthropicThinkingMode maps a normalized thinking level to langchaingo thinking mode.
func anthropicThinkingMode(level string) llms.ThinkingMode {
	switch NormalizeThinkingLevel(level) {
	case ThinkingLevelOff:
		return llms.ThinkingModeNone
	case ThinkingLevelLow:
		return llms.ThinkingModeLow
	case ThinkingLevelMedium:
		return llms.ThinkingModeMedium
	case ThinkingLevelHigh:
		return llms.ThinkingModeHigh
	default:
		return llms.ThinkingModeAuto
	}
}

// thinkingEnabled reports whether extended thinking should be enabled for the session level.
func thinkingEnabled(level string) bool {
	return NormalizeThinkingLevel(level) != ThinkingLevelOff
}

// deepSeekReasoningEffort maps a normalized thinking level to DeepSeek reasoning_effort.
// DeepSeek accepts only high|low|medium|max|xhigh — never "none". Callers must omit
// reasoning_effort when thinking is disabled (ThinkingLevelOff).
func deepSeekReasoningEffort(level string) string {
	switch NormalizeThinkingLevel(level) {
	case ThinkingLevelLow:
		return "low"
	case ThinkingLevelMedium:
		return "medium"
	case ThinkingLevelHigh:
		return "high"
	default:
		// default and off both map to high; off callers should skip this field.
		return "high"
	}
}
