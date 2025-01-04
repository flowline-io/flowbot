package notify

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "notify list",
		Help:   `List notify`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			list, err := store.Database.ListConfigByPrefix(ctx.AsUser, "", "notify:")
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.InfoMsg{
				Title: "Notify",
				Model: list,
			}
		},
	},
	{
		Define: "notify delete [string]",
		Help:   `Delete notify`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			name, _ := tokens[2].Value.String()
			key := fmt.Sprintf("notify:%s", name)
			err := store.Database.ConfigDelete(ctx.AsUser, "", key)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "notify config",
		Help:   `Create notify`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.FormMsg(ctx, createNotifyFormID)
		},
	},
}
