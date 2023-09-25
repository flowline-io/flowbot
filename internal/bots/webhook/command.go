package webhook

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"strings"
	"time"
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
		Define: `list`,
		Help:   `List webhook`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			prefix := "webhook:"
			items, err := store.Chatbot.DataList(ctx.AsUser, ctx.Original, types.DataFilter{Prefix: &prefix})
			if err != nil {
				return nil
			}

			m := make(map[string]interface{})
			for _, item := range items {
				flag := strings.ReplaceAll(item.Key, "webhook:", "")
				m[item.Key] = bots.ServiceURL(ctx, Name, fmt.Sprintf("webhook/%s", flag), nil)
			}

			return types.InfoMsg{
				Title: "Webhook list",
				Model: m,
			}
		},
	},
	{
		Define: `create`,
		Help:   `create webhook`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			p := types.KV{}
			p["uid"] = ctx.AsUser.String()
			flag, err := bots.StoreParameter(p, time.Now().Add(24*365*time.Hour))
			if err != nil {
				return types.TextMsg{Text: "error parameter"}
			}

			err = store.Chatbot.DataSet(ctx.AsUser, ctx.Original,
				fmt.Sprintf("webhook:%s", flag), types.KV{
					"value": "",
				})
			if err != nil {
				return types.TextMsg{Text: "error create"}
			}

			return types.TextMsg{Text: fmt.Sprintf("Webhook: %s", bots.ServiceURL(ctx, Name, fmt.Sprintf("webhook/%s", flag), nil))}
		},
	},
	{
		Define: `del [string]`,
		Help:   `delete webhook`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flag, _ := tokens[1].Value.String()

			err := store.Chatbot.ParameterDelete(flag)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed"}
			}

			err = store.Chatbot.DataDelete(ctx.AsUser, ctx.Original, fmt.Sprintf("webhook:%s", flag))
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
}
