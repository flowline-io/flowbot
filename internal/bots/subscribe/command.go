package subscribe

import (
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/channels"
	"github.com/flowline-io/flowbot/pkg/parser"
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
