package flows

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/action"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/trigger"
)

type RuleRegistry interface {
	FindTrigger(botName, ruleID string) (*trigger.Rule, error)
	FindAction(botName, ruleID string) (*action.Rule, error)
}

type ChatbotRuleRegistry struct{}

func NewChatbotRuleRegistry() RuleRegistry {
	return &ChatbotRuleRegistry{}
}

func (r *ChatbotRuleRegistry) FindTrigger(botName, ruleID string) (*trigger.Rule, error) {
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

func (r *ChatbotRuleRegistry) FindAction(botName, ruleID string) (*action.Rule, error) {
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
