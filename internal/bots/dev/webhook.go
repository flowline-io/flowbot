package dev

import (
	"github.com/flowline-io/flowbot/internal/ruleset/webhook"
	"github.com/flowline-io/flowbot/internal/types"
)

const (
	ExampleWebhookID = "example"
)

var webhookRules = []webhook.Rule{
	{
		Id:     ExampleWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			return types.TextMsg{Text: "hello webhook"}
		},
	},
}
