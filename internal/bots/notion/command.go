package notion

import (
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/parser"
	"github.com/sysatom/flowbot/pkg/providers/notion"
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
			j, err := store.Chatbot.ConfigGet(ctx.AsUser, ctx.Original, "token")
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
				logs.Err.Println(err)
				return types.TextMsg{Text: "import error"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
}