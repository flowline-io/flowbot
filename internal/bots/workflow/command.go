package workflow

import (
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "task run",
		Help:   `Run one task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "todo"}
		},
	},
	{
		Define: "task create",
		Help:   `Create one task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "todo"}
		},
	},
	{
		Define: "task run [number]",
		Help:   `Run task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "todo"}
		},
	},
	{
		Define: "task error",
		Help:   `get workflow step's last error message`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "todo"}
		},
	},
	{
		Define: "task start [number]",
		Help:   `Start task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "todo"}
		},
	},
	{
		Define: "task stop [number]",
		Help:   `Stop task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "todo"}
		},
	},
	{
		Define: "workflow stat",
		Help:   `workflow job statisticians`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "todo"}
		},
	},
	{
		Define: "workflow list",
		Help:   `print workflow list`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return types.TextMsg{Text: "todo"}
		},
	},
}
