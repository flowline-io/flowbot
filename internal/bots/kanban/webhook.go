package kanban

import (
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	json "github.com/json-iterator/go"
	"net/http"
	"strings"
)

const (
	KanbanWebhookID = "kanban"
)

var webhookRules = []webhook.Rule{
	{
		Id:     KanbanWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, method string, data []byte) types.MsgPayload {
			if method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			token, _ := providers.GetConfig(kanboard.ID, kanboard.WebhookTokenKey)
			flog.Debug("kanban token %s", token) // TODO check token

			var resp kanboard.EventResponse
			err := json.Unmarshal(data, &resp)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error event response"}
			}

			// metrics
			go func() {
				stats.KanbanEventTotalCounter(resp.EventName).Inc()
			}()

			switch resp.EventName {
			case "task.close":
				var result kanboard.TaskClose
				err = unmarshal(resp.EventData, &result)
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error unmarshal"}
				}

				if result.Task.Reference == "" {
					return nil
				}

				s := strings.Split(result.Task.Reference, ":")
				if len(s) != 2 {
					return nil
				}

				switch s[0] {
				case hoarder.ID:
					err = event.BotEventFire(ctx.Context(), types.BookmarkArchiveBotEventID, types.BotEvent{
						Uid:   ctx.AsUser.String(),
						Topic: ctx.Topic,
						Param: types.KV{
							"id": s[1],
						},
					})
					if err != nil {
						flog.Error(err)
						return types.TextMsg{Text: "error bookmark archive"}
					}
				}
			}

			return nil
		},
	},
}
