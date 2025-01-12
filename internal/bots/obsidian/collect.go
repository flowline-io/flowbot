package obsidian

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/collect"
)

const (
	StatsCollectID = "obsidian_stats_collect"
)

var collectRules = []collect.Rule{
	{
		Id:   StatsCollectID,
		Help: "import obsidian stats",
		Args: []string{"data"},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			j := types.KV{}
			err := j.Scan(content)
			if err != nil {
				return nil
			}
			if err != nil {
				return nil
			}
			html, _ := j.String("data")
			if html == "" {
				return nil
			}
			_ = store.Database.DataSet(ctx.AsUser, ctx.Topic, "getCollectionStatsHTML", types.KV{
				"value": html,
			})
			return nil
		},
	},
}
