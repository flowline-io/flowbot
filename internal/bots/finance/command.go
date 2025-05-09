package finance

import (
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/wallos"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: `wallos`,
		Help:   `Get wallos subscriptions`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			client := wallos.GetClient()
			list, err := client.GetSubscriptions(ctx.Context())
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.InfoMsg{
				Title: "Wallos Subscriptions",
				Model: list,
			}
		},
	},
	{
		Define: "bill",
		Help:   `Import bill`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return chatbot.FormMsg(ctx, importBillFormID)
		},
	},
}
