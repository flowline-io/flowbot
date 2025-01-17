package kanban

import (
	"net/http"
	"strings"

	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	json "github.com/json-iterator/go"
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
			case kanboard.TaskCloseEvent:
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
				app := s[0]
				id := s[1]

				switch app {
				case hoarder.ID:
					err = event.BotEventFire(ctx, types.BookmarkArchiveBotEventID, types.KV{
						"id": id,
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
