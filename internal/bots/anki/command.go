package anki

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/parser"
)

var commandRules = []command.Rule{
	{
		Define: "stats",
		Help:   `Anki collection statistics`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			j, err := store.Database.DataGet(ctx.AsUser, ctx.Topic, "getCollectionStatsHTML")
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
