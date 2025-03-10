package dev

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
)

const (
	ExampleWebhookID = "example"
)

var webhookRules = []webhook.Rule{
	{
		Id:     ExampleWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			return types.TextMsg{Text: fmt.Sprintf("%s %s", ctx.Method, string(data))}
		},
	},
}
