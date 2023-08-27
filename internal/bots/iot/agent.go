package iot

import (
	"github.com/sysatom/flowbot/internal/ruleset/agent"
	"github.com/sysatom/flowbot/internal/types"
)

const (
	AgentVersion   = 1
	ExampleAgentID = "iot_example_agent"
)

var agentRules = []agent.Rule{
	{
		Id: ExampleAgentID,
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			return nil
		},
	},
}
