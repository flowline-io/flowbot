package gitea

import (
	"encoding/json"
	"github.com/flowline-io/flowbot/internal/ruleset/webhook"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
)

const (
	IssueWebhookID = "issue"
)

var webhookRules = []webhook.Rule{
	{
		Id:     IssueWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			body, ok := content.Any("body")
			if !ok {
				return types.TextMsg{Text: "error"}
			}

			var issue gitea.Issue
			err := json.Unmarshal(body.([]byte), &issue)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error"}
			}

			flog.Info("action %s issue %s", issue.Action, issue.Issue.Title)

			return types.TextMsg{Text: "issue webhook"}
		},
	},
}
