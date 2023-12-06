package gpt

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/utils"
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
			v, err := store.Database.ConfigGet(ctx.AsUser, ctx.Original, ApiKey)
			if err != nil {
				flog.Error(err)
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
			v, err := store.Database.ConfigGet(ctx.AsUser, ctx.Original, ApiKey)
			if err != nil {
				flog.Error(err)
			}
			old, _ := v.String("value")

			// set
			err = store.Database.ConfigSet(ctx.AsUser, ctx.Original, ApiKey, types.KV{
				"value": key,
			})
			if err != nil {
				flog.Error(err)
			}

			return types.TextMsg{Text: fmt.Sprintf("%s --> %s", old, key)}
		},
	},
}
