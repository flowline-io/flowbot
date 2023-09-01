package dev

import (
	"github.com/flowline-io/flowbot/internal/ruleset/agent"
	"github.com/flowline-io/flowbot/internal/types"
)

const (
	AgentVersion  = 1
	ImportAgentID = "import_agent"
)

var agentRules = []agent.Rule{
	{
		Id:   ImportAgentID,
		Help: "agent example",
		Args: []string{},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			return nil
		},
	},
}
