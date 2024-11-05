package gitea

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/webhook"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/utils"
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

const (
	IssueWebhookID = "issue"
)

var webhookRules = []webhook.Rule{
	{
		Id:     IssueWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, method string, data []byte) types.MsgPayload {
			if method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			var issue *gitea.IssuePayload
			err := jsoniter.Unmarshal(data, &issue)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error"}
			}

			flog.Info("[gitea] issue webhook, method: %s, action: %s", method, issue.Action)
			utils.PrettyPrint(issue)

			switch issue.Action {
			case gitea.HookIssueCreated:
				hookIssueCreated(ctx, issue)
			case gitea.HookIssueOpened:
				hookIssueOpened(ctx, issue)
			case gitea.HookIssueClosed:
			case gitea.HookIssueReOpened:
			case gitea.HookIssueAssigned:
			case gitea.HookIssueUnassigned:
			case gitea.HookIssueLabelUpdated:
			case gitea.HookIssueLabelCleared:
			case gitea.HookIssueMilestoned:
			case gitea.HookIssueDemilestoned:
			default:
				return types.TextMsg{Text: "error action"}
			}

			return types.TextMsg{Text: "issue webhook"}
		},
	},
}
