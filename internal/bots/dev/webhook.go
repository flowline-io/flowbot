package dev

import (
	"fmt"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/internal/agents"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	json "github.com/json-iterator/go"
)

const (
	ExampleWebhookID = "example"
	ChatWebhookID    = "chat"
)

var webhookRules = []webhook.Rule{
	{
		Id:     ExampleWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			return types.TextMsg{Text: fmt.Sprintf("%s %s", ctx.Method, string(data))}
		},
	},
	{
		Id:     ChatWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			if ctx.Method != http.MethodPost {
				return types.TextMsg{Text: "error method"}
			}

			var param struct {
				Text string `json:"text"`
				Ip   string `json:"ip"`
			}
			err := json.Unmarshal(data, &param)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error params"}
			}

			if param.Text == "" {
				return types.TextMsg{Text: "empty text"}
			}

			if param.Ip == "" {
				return types.TextMsg{Text: "Forbidden"}
			}

			// run agent
			go func() {
				tools, err := bots.AvailableTools(ctx)
				if err != nil {
					flog.Error(err)
					return
				}
				ctx.SetTimeout(10 * time.Minute)
				agent, err := agents.ReactAgent(ctx.Context(), tools)
				if err != nil {
					flog.Error(err)
					return
				}

				messages, err := agents.DefaultTemplate().Format(ctx.Context(), map[string]any{
					"content": param.Text,
				})
				if err != nil {
					flog.Error(err)
					return
				}

				resp, err := agent.Generate(ctx.Context(), messages)
				if err != nil {
					flog.Error(err)
					return
				}

				if resp != nil && resp.Content != "" {
					err = event.SendMessage(ctx, types.TextMsg{Text: resp.Content})
					if err != nil {
						flog.Error(err)
						return
					}
				}
			}()

			return types.TextMsg{Text: "ok"}
		},
	},
}
