package clipboard

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/parser"
)

var commandRules = []command.Rule{
	{
		Define: "share [string]",
		Help:   `share clipboard to agent`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			txt, _ := tokens[1].Value.String()
			data := types.KV{}
			data["txt"] = txt
			return bots.InstructMsg(ctx, ShareInstruct, data)
		},
	},
}
