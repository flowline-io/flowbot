package notify

import (
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name:  "notify_example",
		Scope: cron.CronScopeSystem,
		When:  "* * * * *",
		Action: func(types.Context) []types.MsgPayload {
			return nil
		},
	},
}
