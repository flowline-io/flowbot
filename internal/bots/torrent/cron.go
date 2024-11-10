package torrent

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/cron"
)

var cronRules = []cron.Rule{
	{
		Name: "torrent_clear",
		When: "*/10 * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			err := torrentClear(ctx.Context())
			if err != nil {
				flog.Error(err)
			}
			return nil
		},
	},
}
