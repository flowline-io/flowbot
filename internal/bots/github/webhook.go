package github

import (
	"fmt"
	"net/http"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/drone"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
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
				client, err := gitea.GetClient()
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error"}
				}

				// get namespace
				user, err := client.GetMyUserInfo()
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error"}
				}

				// create build
				dClient := drone.GetClient()
				build, err := dClient.CreateBuild(user.LoginName, drone.DefaultDeployRepoName)
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error"}
				}

				// send message
				err = event.SendMessage(ctx, types.TextMsg{Text: fmt.Sprintf("%s/%d", config.App.Search.UrlBaseMap[drone.ID], build.ID)})
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error"}
				}

				return types.TextMsg{Text: "deploy"}
			default:
				return types.TextMsg{Text: "upnot supported"}
			}
		},
	},
}
