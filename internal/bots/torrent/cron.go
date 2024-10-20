package torrent

import (
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/types"
)

var cronRules = []cron.Rule{
	{
		Name: "torrent_clear",
		When: "0 * * * *",
		Action: func(types.Context) []types.MsgPayload {
			return nil
		},
	},
}
