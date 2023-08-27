package dev

import (
	"github.com/sysatom/flowbot/internal/ruleset/agent"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
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
			logs.Info.Println(content)
			return nil
		},
	},
}
