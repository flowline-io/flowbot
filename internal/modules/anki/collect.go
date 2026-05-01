package anki

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/collect"
)

const (
	StatsCollectID  = "stats_collect"
	ReviewCollectID = "review_collect"
)

var collectRules = []collect.Rule{
	{
		Id:   StatsCollectID,
		Help: "import anki stats",
		Args: []string{"html"},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			j := types.KV{}
			err := j.Scan(content)
			if err != nil {
				return nil
			}
			html, _ := j.String("html")
			if html == "" {
				return nil
			}
			_ = store.Database.DataSet(ctx.AsUser, ctx.Topic, "getCollectionStatsHTML", types.KV{
				"value": html,
			})
			return nil
		},
	},
	{
		Id:   ReviewCollectID,
		Help: "import anki review count",
		Args: []string{"num"},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			j := types.KV{}
			err := j.Scan(content)
			if err != nil {
				return nil
			}
			num, _ := j.Int64("num")
			_ = store.Database.DataSet(ctx.AsUser, ctx.Topic, "getNumCardsReviewedToday", types.KV{
				"value": num,
			})
			return nil
		},
	},
}
