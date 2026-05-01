package reader

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
)

const (
	MinifluxWebhookID = "miniflux"
)

var webhookRules = []webhook.Rule{
	{
		Id:     MinifluxWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			if ctx.Method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			var resp miniflux.WebhookEvent
			err := sonic.Unmarshal(data, &resp)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error event response"}
			}

			switch resp.EventType {
			case miniflux.NewEntriesEventType:

			case miniflux.SaveEntryEventType:
				err = event.BotEventFire(ctx, types.BookmarkCreateBotEventID, types.KV{
					"url": resp.Entry.URL,
				})
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error bookmark create"}
				}
			}

			return nil
		},
	},
}
