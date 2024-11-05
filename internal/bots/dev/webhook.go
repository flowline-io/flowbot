package dev

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/webhook"
)

const (
	ExampleWebhookID = "example"
)

var webhookRules = []webhook.Rule{
	{
		Id:     ExampleWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, method string, data []byte) types.MsgPayload {
			return types.TextMsg{Text: fmt.Sprintf("%s %s", method, string(data))}
		},
	},
}
