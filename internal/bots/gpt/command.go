package gpt

import (
	"fmt"
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/parser"
	"github.com/sysatom/flowbot/pkg/utils"
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
		Define: "key",
		Help:   `get api key`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			// get
			v, err := store.Chatbot.ConfigGet(ctx.AsUser, ctx.Original, ApiKey)
			if err != nil {
				logs.Err.Println("bot command key", err)
			}
			key, _ := v.String("value")

			return types.TextMsg{Text: fmt.Sprintf("key: %s", utils.Masker(key, 3))}
		},
	},
	{
		Define: "key [string]",
		Help:   `Set api key`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			key, _ := tokens[1].Value.String()

			// get
			v, err := store.Chatbot.ConfigGet(ctx.AsUser, ctx.Original, ApiKey)
			if err != nil {
				logs.Err.Println("bot command key [string]", err)
			}
			old, _ := v.String("value")

			// set
			err = store.Chatbot.ConfigSet(ctx.AsUser, ctx.Original, ApiKey, types.KV{
				"value": key,
			})
			if err != nil {
				logs.Err.Println("bot command key [string]", err)
			}

			return types.TextMsg{Text: fmt.Sprintf("%s --> %s", old, key)}
		},
	},
}
