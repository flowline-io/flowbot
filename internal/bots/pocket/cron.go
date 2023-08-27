package pocket

import (
	"errors"
	"github.com/sysatom/flowbot/internal/ruleset/cron"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/providers/pocket"
	"gorm.io/gorm"
)

var cronRules = []cron.Rule{
	{
		Name: "pocket_add",
		When: "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			oauth, err := store.Chatbot.OAuthGet(ctx.AsUser, ctx.Original, Name)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				logs.Err.Println("bot command pocket oauth", err)
			}
			if oauth.Token == "" {
				return nil
			}

			provider := pocket.NewPocket(Config.ConsumerKey, "", "", oauth.Token)
			items, err := provider.Retrieve(10)
			if err != nil {
				logs.Err.Println(err)
				return nil
			}

			var r []types.MsgPayload
			for _, item := range items.List {
				r = append(r, types.LinkMsg{
					Title: item.GivenTitle,
					Url:   item.GivenUrl,
				})
			}
			return r
		},
	},
}
