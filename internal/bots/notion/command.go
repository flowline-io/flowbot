package notion

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/notion"
)

var commandRules = []command.Rule{
	{
		Define: "setting",
		Help:   `Bot setting`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.SettingMsg(ctx, Name)
		},
	},
	{
		Define: "search [string]",
		Help:   "Searches all original pages, databases, and child pages/databases that are shared with the integration.",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			query, _ := tokens[1].Value.String()

			// token value
			j, err := store.Database.ConfigGet(ctx.AsUser, ctx.Original, "token")
			if err != nil {
				return nil
			}
			token, _ := j.String("value")
			if token == "" {
				return types.TextMsg{Text: "set config"}
			}

			provider := notion.NewNotion(token)
			pages, err := provider.Search(query)
			if err != nil {
				return types.TextMsg{Text: "search error"}
			}
			if len(pages) == 0 {
				return types.TextMsg{Text: "Empty"}
			}
			var links types.LinkListMsg
			for _, page := range pages {
				links.Links = append(links.Links, types.LinkMsg{Title: page.Object, Url: page.URL})
			}
			return links
		},
	},
	{
		Define: "import [string]",
		Help:   "Append to MindCache page",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			text, _ := tokens[1].Value.String()

			// token value
			j, err := bots.SettingGet(ctx, Name, tokenSettingKey)
			if err != nil {
				return nil
			}
			token, _ := j.StringValue()
			if token == "" {
				return types.TextMsg{Text: "set config"}
			}

			// block id
			j2, err := bots.SettingGet(ctx, Name, importPageIdSettingKey)
			if err != nil {
				return nil
			}
			pageId, _ := j2.StringValue()
			if pageId == "" {
				return types.TextMsg{Text: "set config"}
			}

			provider := notion.NewNotion(token)
			err = provider.AppendBlock(pageId, text)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "import error"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
}
