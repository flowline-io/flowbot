package flows

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/action"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/trigger"
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

func findTriggerRule(botName, ruleID string) (*trigger.Rule, error) {
	bots := chatbot.List()
	b := bots[botName]
	if b == nil {
		if botName == "system" {
			b = bots["dev"]
			botName = "dev"
		}
		if b == nil {
			return nil, fmt.Errorf("bot not found: %s", botName)
		}
	}
	for _, rs := range b.Rules() {
		switch v := rs.(type) {
		case []trigger.Rule:
			for i := range v {
				if v[i].Id == ruleID {
					return &v[i], nil
				}
			}
		}
	}
	return nil, fmt.Errorf("trigger rule not found: %s/%s", botName, ruleID)
}

func findActionRule(botName, ruleID string) (*action.Rule, error) {
	bots := chatbot.List()
	b := bots[botName]
	if b == nil {
		if botName == "system" {
			b = bots["dev"]
			botName = "dev"
		}
		if b == nil {
			return nil, fmt.Errorf("bot not found: %s", botName)
		}
	}
	for _, rs := range b.Rules() {
		switch v := rs.(type) {
		case []action.Rule:
			for i := range v {
				if v[i].Id == ruleID {
					return &v[i], nil
				}
			}
		}
	}
	return nil, fmt.Errorf("action rule not found: %s/%s", botName, ruleID)
}
