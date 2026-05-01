package dev

import (
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name:  "dev_demo",
		Help:  "cron example",
		Scope: cron.CronScopeSystem,
		When:  "0 */10 * * *",
		Action: func(types.Context) []types.MsgPayload {
			return nil
		},
	},
}
