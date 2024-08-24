package pocket

import (
	"errors"
	"time"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/pocket"
	"gorm.io/gorm"
)

var commandRules = []command.Rule{
	{
		Define: "oauth",
		Help:   `OAuth`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			// check oauth token
			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, Name)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				flog.Error(err)
			}
			if oauth.Token != "" {
				return types.TextMsg{Text: "App is authorized"}
			}

			flag, err := bots.StoreParameter(types.KV{
				"uid":   ctx.AsUser.String(),
				"topic": ctx.Topic,
			}, time.Now().Add(time.Hour))
			if err != nil {
				flog.Error(err)
				return nil
			}
			key, _ := providers.GetConfig(pocket.ID, pocket.ClientIdKey)
			redirectURI := providers.RedirectURI(pocket.ID, flag)
			provider := pocket.NewPocket(key.String(), "", redirectURI, "")
			_, err = provider.GetCode("")
			if err != nil {
				return types.TextMsg{Text: "get code error"}
			}
			return types.LinkMsg{Title: "OAuth", Url: provider.GetAuthorizeURL()}
		},
	},
	{
		Define: "list",
		Help:   `newest 10`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			oauth, err := store.Database.OAuthGet(ctx.AsUser, ctx.Topic, Name)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				flog.Error(err)
			}
			if oauth.Token == "" {
				return types.TextMsg{Text: "App is unauthorized"}
			}

			key, _ := providers.GetConfig(pocket.ID, pocket.ClientIdKey)
			provider := pocket.NewPocket(key.String(), "", "", oauth.Token)
			items, err := provider.Retrieve(10)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "retrieve error"}
			}

			var header []string
			var row [][]interface{}
			if len(items.List) > 0 {
				header = []string{"Id", "GivenUrl", "GivenTitle", "WordCount"}
				for _, v := range items.List {
					row = append(row, []interface{}{v.Id, v.GivenUrl, v.GivenTitle, v.WordCount})
				}
			}

			return bots.StorePage(ctx, model.PageTable, "Newest List", types.TableMsg{
				Title:  "Newest List",
				Header: header,
				Row:    row,
			})
		},
	},
}
