package subscribe

import (
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/channels"
	"github.com/sysatom/flowbot/pkg/parser"
)

var commandRules = []command.Rule{
	{
		Define: "info",
		Help:   `Bot info`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return nil
		},
	},
	{
		Define: "list",
		Help:   `List subscribe`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.InfoMsg{
				Title: "Subscribes",
				Model: channels.List(),
			}
		},
	},
}
