package permission

import (
	"strings"
)

// SuggestedPattern builds a safe always-allow pattern for one permission decision.
// The second return value is false when always should not be offered.
func SuggestedPattern(key, primary string, bash ParseBashCommand) (string, bool) {
	primary = strings.TrimSpace(primary)
	if primary == "" {
		return "", false
	}
	switch key {
	case "bash":
		if bash.Complex || bash.HasChain {
			return "", false
		}
		prefix := strings.TrimSpace(bash.Prefix)
		if prefix == "" {
			prefix = primary
		}
		if IsOverlyBroadPattern(prefix) {
			return "", false
		}
		if strings.Contains(prefix, " ") {
			rest := strings.TrimSpace(strings.TrimPrefix(primary, prefix))
			if rest == "" {
				return prefix + "*", true
			}
			return prefix + " *", true
		}
		return prefix + "*", true
	case "read", "edit":
		pattern := ParentDirPattern(primary)
		if pattern == "" || IsOverlyBroadPattern(pattern) {
			return "", false
		}
		return pattern, true
	case "websearch", "skill", KeyKnowledge:
		if IsOverlyBroadPattern(primary) {
			return "", false
		}
		return primary, true
	default:
		return "", false
	}
}
