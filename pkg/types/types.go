package types

import (
	"github.com/flowline-io/flowbot/pkg/config"
)

func AppUrl() string {
	return config.App.Flowbot.URL
}

type Ruler interface {
	ID() string
	TYPE() RulesetType
}

type RulesetType string

const (
	ActionRule     RulesetType = "action"
	CommandRule    RulesetType = "command"
	FormRule       RulesetType = "form"
	TriggerRule    RulesetType = "trigger"
	WebserviceRule RulesetType = "webservice"
	WorkflowRule   RulesetType = "workflow"
)
