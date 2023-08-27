package clipboard

import (
	"github.com/flowline-io/flowbot/internal/ruleset/agent"
	"github.com/flowline-io/flowbot/internal/types"
)

const (
	AgentVersion  = 1
	UploadAgentID = "clipboard_upload"
)

var agentRules = []agent.Rule{
	{
		Id:   UploadAgentID,
		Help: "update clipboard",
		Args: []string{"txt"},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			j := types.KV{}
			err := j.Scan(content)
			if err != nil {
				return nil
			}
			txt, ok := j.String("txt")
			if !ok {
				return nil
			}
			return types.TextMsg{Text: txt}
		},
	},
}
