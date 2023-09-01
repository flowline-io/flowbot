package pocket

import (
	"errors"
	"github.com/flowline-io/flowbot/internal/ruleset/cron"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/pocket"
	"gorm.io/gorm"
)

var cronRules = []cron.Rule{
	{
		Name: "pocket_add",
		When: "* * * * *",
		Action: func(ctx types.Context) []types.MsgPayload {
			oauth, err := store.Chatbot.OAuthGet(ctx.AsUser, ctx.Original, Name)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				flog.Error(err)
			}
			if oauth.Token == "" {
				return nil
			}

			provider := pocket.NewPocket(Config.ConsumerKey, "", "", oauth.Token)
			items, err := provider.Retrieve(10)
			if err != nil {
				flog.Error(err)
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
