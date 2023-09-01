package server

import (
	"github.com/flowline-io/flowbot/internal/ruleset/agent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	AgentVersion = 1
	StatsAgentID = "stats_agent"
)

var agentRules = []agent.Rule{
	{
		Id:   StatsAgentID,
		Help: "upload server status",
		Args: []string{"cpu", "memory", "info"},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			j := types.KV{}
			err := j.Scan(content)
			if err != nil {
				return nil
			}
			// alert

			// store
			err = store.Chatbot.DataSet(ctx.AsUser, ctx.Original, "stats", j)
			if err != nil {
				flog.Error(err)
				return nil
			}
			return nil
		},
	},
}
