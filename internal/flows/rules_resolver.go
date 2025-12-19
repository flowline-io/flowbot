package flows

import (
	"strings"
)

func normalizeFlowTriggerType(triggerType string) (botName, ruleID string) {
	if strings.Contains(triggerType, "|") {
		parts := strings.SplitN(triggerType, "|", 2)
		return parts[0], parts[1]
	}
	if triggerType == "" {
		return "", ""
	}
	return "dev", triggerType
}
