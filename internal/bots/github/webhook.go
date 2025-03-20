package github

import (
	"net/http"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
)

const (
	PackageWebhookID = "package"
)

var webhookRules = []webhook.Rule{
	{
		Id:     PackageWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			if ctx.Method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			events, ok := ctx.Headers["X-GitHub-Event"]
			if !ok {
				return types.TextMsg{Text: "error header"}
			}
			if len(events) == 0 {
				return types.TextMsg{Text: "error event"}
			}

			switch events[0] {
			case "ping":
				return types.TextMsg{Text: "pong"}
			case "package":
				err := deploy(ctx)
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error deploy"}
				}

				return types.TextMsg{Text: "deploy"}
			default:
				return types.TextMsg{Text: "upnot supported"}
			}
		},
	},
}
