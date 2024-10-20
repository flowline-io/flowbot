package torrent

import (
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
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
