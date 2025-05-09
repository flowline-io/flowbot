package gitea

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	IssueWebhookID = "issue"
	RepoWebhookID  = "repo"
)

var webhookRules = []webhook.Rule{
	{
		Id:     IssueWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			if ctx.Method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			var issue *gitea.IssuePayload
			err := sonic.Unmarshal(data, &issue)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error"}
			}

			flog.Info("[gitea] issue webhook, method: %s, action: %s", ctx.Method, issue.Action)
			utils.PrettyPrintYamlStyle(issue)

			switch issue.Action {
			case gitea.HookIssueCreated:
				err = hookIssueCreated(ctx, issue)
			case gitea.HookIssueOpened:
				err = hookIssueOpened(ctx, issue)
			case gitea.HookIssueClosed:
				err = hookIssueClosed(ctx, issue)
			default:
				return types.TextMsg{Text: "error action"}
			}

			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "done"}
		},
	},
	{
		Id:     RepoWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			if ctx.Method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			eventTypeList, ok := ctx.Headers["X-Gitea-Event"]
			if !ok {
				flog.Warn("[gitea] repo webhook, error header %v", ctx.Headers)
				return types.TextMsg{Text: "error header"}
			}

			if len(eventTypeList) == 0 {
				flog.Warn("[gitea] repo webhook, error header %v", ctx.Headers)
				return types.TextMsg{Text: "error header"}
			}

			eventType := eventTypeList[0]

			var err error
			switch eventType {
			case "push":
				var repoPayload *gitea.RepoPayload
				err := sonic.Unmarshal(data, &repoPayload)
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "error"}
				}

				err = hookPush(ctx, repoPayload)
			default:
				return types.TextMsg{Text: "error event type"}
			}

			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "done"}
		},
	},
}
