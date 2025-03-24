package types

import (
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/lithammer/shortuuid/v4"
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
	CollectRule    RulesetType = "collect"
	CommandRule    RulesetType = "command"
	CronRule       RulesetType = "cron"
	EventRule      RulesetType = "event"
	FormRule       RulesetType = "form"
	InstructRule   RulesetType = "instruct"
	PageRule       RulesetType = "page"
	SettingRule    RulesetType = "setting"
	ToolRule       RulesetType = "tool"
	WebhookRule    RulesetType = "webhook"
	WebserviceRule RulesetType = "webservice"
	WorkflowRule   RulesetType = "workflow"
)
