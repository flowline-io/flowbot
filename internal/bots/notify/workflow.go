package notify

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/workflow"
)

const (
	sendWorkflowActionID = "send"
)

var workflowRules = []workflow.Rule{
	{
		Id:           sendWorkflowActionID,
		Title:        "Send",
		Desc:         "Send notification",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			channel, _ := input.String("channel")
			if channel == "" {
				return nil, fmt.Errorf("%s step, empty channel", sendWorkflowActionID)
			}
			title, _ := input.String("title")
			body, _ := input.String("body")
			if body == "" {
				return nil, fmt.Errorf("%s step, empty body", sendWorkflowActionID)
			}
			url, _ := input.String("url")

			err := notify.ChannelSend(ctx.AsUser, channel, notify.Message{
				Title: title,
				Body:  body,
				Url:   url,
			})
			return types.KV{}, err
		},
	},
}
