package finance

import (
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "bill",
		Help:   `Import bill`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return chatbot.FormMsg(ctx, importBillFormID)
		},
	},
}
