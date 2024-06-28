package webhook

import (
	"errors"
	"fmt"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/ruleset/webhook"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/utils"
	"gorm.io/gorm"
)

var commandRules = []command.Rule{
	{
		Define: `webhook list`,
		Help:   `List webhook`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {

			items, err := store.Database.ListWebhook(ctx.AsUser)
			if err != nil {
				return nil
			}

			m := make(map[string]interface{})
			for _, item := range items {
				m[item.Flag] = fmt.Sprintf("%s/webhook/%s?secret=%s [%s] (%d)",
					types.AppUrl(), item.Flag, item.Secret, stateStr(item.State), item.TriggerCount)
			}

			return types.InfoMsg{
				Title: "Webhook list",
				Model: m,
			}
		},
	},
	{
		Define: `webhook create [flag]`,
		Help:   `create webhook`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flag, _ := tokens[2].Value.String()

			var webhookRule webhook.Rule
			var botHandler bots.Handler
			for _, handler := range bots.List() {
				for _, item := range handler.Rules() {
					switch v := item.(type) {
					case []webhook.Rule:
						for _, rule := range v {
							if rule.Id == flag {
								botHandler = handler
								webhookRule = rule
							}
						}
					}
				}
			}

			if botHandler == nil {
				return types.TextMsg{Text: "not found"}
			}

			if !webhookRule.Secret {
				return types.TextMsg{Text: "not need create"}
			}

			// find exist webhook
			find, err := store.Database.GetWebhookByUidAndFlag(ctx.AsUser, flag)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				flog.Error(err)
				return types.TextMsg{Text: "find failed"}
			}
			if find != nil {
				return types.TextMsg{Text: find.Secret}
			}

			secret, err := utils.GenerateRandomString(32)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "generate secret failed"}
			}

			_, err = store.Database.CreateWebhook(&model.Webhook{
				UID:          ctx.AsUser.String(),
				Topic:        ctx.RcptTo,
				Flag:         flag,
				Secret:       secret,
				TriggerCount: 0,
				State:        model.WebhookActive,
			})
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "create failed"}
			}

			return types.TextMsg{Text: secret}
		},
	},
	{
		Define: `webhook del [secret]`,
		Help:   `delete webhook`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			secret, _ := tokens[2].Value.String()

			find, err := store.Database.GetWebhookBySecret(secret)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "find failed"}
			}

			if find.UID != ctx.AsUser.String() {
				return types.TextMsg{Text: "auth failed"}
			}

			err = store.Database.DeleteWorkflow(find.ID)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "delete failed"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: `webhook activate [secret]`,
		Help:   `activate webhook`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			secret, _ := tokens[2].Value.String()

			find, err := store.Database.GetWebhookBySecret(secret)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "find failed"}
			}

			if find.UID != ctx.AsUser.String() {
				return types.TextMsg{Text: "auth failed"}
			}

			find.State = model.WebhookActive
			err = store.Database.UpdateWebhook(find)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "update failed"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: `webhook inactive [secret]`,
		Help:   `inactive webhook`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			secret, _ := tokens[2].Value.String()

			find, err := store.Database.GetWebhookBySecret(secret)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "find failed"}
			}

			if find.UID != ctx.AsUser.String() {
				return types.TextMsg{Text: "auth failed"}
			}

			find.State = model.WebhookInactive
			err = store.Database.UpdateWebhook(find)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "update failed"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
}
