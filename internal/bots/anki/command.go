package anki

import (
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
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
		Define: "stats",
		Help:   `Anki collection statistics`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			j, err := store.Chatbot.DataGet(ctx.AsUser, ctx.Original, "getCollectionStatsHTML")
			if err != nil {
				return types.TextMsg{Text: "Empty"}
			}
			html, ok := j.String("value")
			if !ok {
				return types.TextMsg{Text: "Empty"}
			}
			return bots.StorePage(ctx, model.PageHtml, "Anki collection statistics", types.HtmlMsg{Raw: html})
		},
	},
}
