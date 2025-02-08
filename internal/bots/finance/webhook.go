package finance

import (
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/flowline-io/flowbot/pkg/utils"
	"net/http"
)

const (
	WallosWebhookID = "wallos"
)

var webhookRules = []webhook.Rule{
	{
		Id:     WallosWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, method string, data []byte) types.MsgPayload {
			if method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			err := event.SendMessage(ctx, types.TextMsg{Text: utils.BytesToString(data)})
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: err.Error()}
			}

			return nil
		},
	},
}
