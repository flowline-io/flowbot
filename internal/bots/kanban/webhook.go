package kanban

import (
	"net/http"

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

			var resp kanboard.EventResponse
			err := json.Unmarshal(data, &resp)
			if err != nil {
				return types.TextMsg{Text: "error event response"}
			}

			// metrics
			stats.KanbanEventTotalCounter(resp.EventName).Inc()

			return types.TextMsg{Text: "kanban webhook"}
		},
	},
}
