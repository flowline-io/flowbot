package github

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	json "github.com/json-iterator/go"
	"net/http"
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

			events, ok := ctx.Headers["X-Github-Event"]
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
				var a github.PackageWebhook
				err := json.Unmarshal(data, &a)
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error unmarshal"}
				}
				if a.Package.PackageVersion.ContainerMetadata.Tag.Name != "latest" {
					flog.Info("ignore package tag %s digest %s", a.Package.PackageVersion.ContainerMetadata.Tag.Name,
						a.Package.PackageVersion.ContainerMetadata.Tag.Digest)
					return types.TextMsg{Text: "not latest"}
				}

				if a.Action != "published" {
					flog.Info("ignore package action %s", a.Action)
					return types.TextMsg{Text: "not published"}
				}

				err = deploy(ctx)
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error deploy"}
				}

				return types.TextMsg{Text: "deploy"}
			default:
				return types.TextMsg{Text: "not supported"}
			}
		},
	},
}
