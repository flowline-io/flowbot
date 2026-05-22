package types

import (
	"github.com/lithammer/shortuuid/v4"

	"github.com/flowline-io/flowbot/pkg/config"
)

func Id() string {
	return shortuuid.New()
}

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
	PageRule       RulesetType = "page"
	TriggerRule    RulesetType = "trigger"
	WebserviceRule RulesetType = "webservice"
	WorkflowRule   RulesetType = "workflow"
)
