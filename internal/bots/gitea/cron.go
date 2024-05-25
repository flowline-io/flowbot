package gitea

import (
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/types"
)

var cronRules = []cron.Rule{
	{
		Name: "gitea_example",
		When: "* * * * *",
		Action: func(types.Context) []types.MsgPayload {
			return nil
		},
	},
}
