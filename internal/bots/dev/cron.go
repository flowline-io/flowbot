package dev

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name: "dev_demo",
		Help: "cron example",
		When: "0 */1 * * *",
		Action: func(types.Context) []types.MsgPayload {
			return nil
		},
	},
}
