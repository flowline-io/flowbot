package anki

import (
	"github.com/sysatom/flowbot/internal/ruleset/agent"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/types"
)

const (
	AgentVersion  = 1
	StatsAgentID  = "stats_agent"
	ReviewAgentID = "review_agent"
)

var agentRules = []agent.Rule{
	{
		Id:   StatsAgentID,
		Help: "import anki stats",
		Args: []string{"html"},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			j := types.KV{}
			err := j.Scan(content)
			if err != nil {
				return nil
			}
			if err != nil {
				return nil
			}
			html, _ := j.String("html")
			if html == "" {
				return nil
			}
			_ = store.Chatbot.DataSet(ctx.AsUser, ctx.Original, "getCollectionStatsHTML", types.KV{
				"value": html,
			})
			return nil
		},
	},
	{
		Id:   ReviewAgentID,
		Help: "import anki review count",
		Args: []string{"num"},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			j := types.KV{}
			err := j.Scan(content)
			if err != nil {
				return nil
			}
			if err != nil {
				return nil
			}
			num, _ := j.Int64("num")
			_ = store.Chatbot.DataSet(ctx.AsUser, ctx.Original, "getNumCardsReviewedToday", types.KV{
				"value": num,
			})
			return nil
		},
	},
}
