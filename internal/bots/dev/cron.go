package dev

import (
	"github.com/sysatom/flowbot/internal/ruleset/cron"
	"github.com/sysatom/flowbot/internal/types"
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
