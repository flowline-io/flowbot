package gitea

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webhook"
	"github.com/flowline-io/flowbot/internal/types"
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

			var issue gitea.IssuePayload
			err := jsoniter.Unmarshal(data, &issue)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error"}
			}

			utils.PrettyPrint(issue)

			return types.TextMsg{Text: "issue webhook"}
		},
	},
}
