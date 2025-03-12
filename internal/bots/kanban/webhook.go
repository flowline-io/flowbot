package kanban

import (
	"net/http"
	"strings"

	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
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
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			if ctx.Method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			var resp kanboard.EventResponse
			err := json.Unmarshal(data, &resp)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error event response"}
			}

			flog.Info("[kanban] webhook event: %s", resp.EventName)

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
				var app, category, id string
				switch len(s) {
				case 2:
					app = s[0]
					id = s[1]
				case 3:
					app = s[0]
					category = s[1]
					id = s[2]
				default:
					flog.Warn("invalid kanban take reference %s", result.Task.Reference)
					return nil
				}

				switch app {
				case hoarder.ID:
					err = event.BotEventFire(ctx, types.BookmarkArchiveBotEventID, types.KV{
						"id": id,
					})
					if err != nil {
						flog.Error(err)
						return types.TextMsg{Text: "error bookmark archive"}
					}
				case gitea.ID:
					if category == "commit" {
						flog.Info("commit review done: commit id %s", id)
					}
				}
			}

			return nil
		},
	},
}
