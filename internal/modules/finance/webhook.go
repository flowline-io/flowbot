package finance

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
)

const (
	WallosWebhookID = "wallos"
)

type wallosWebhookData struct {
	Event     string `json:"event"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	Category  string `json:"category"`
	Payee     string `json:"payee"`
	Notes     string `json:"notes"`
	Date      string `json:"date"`
	Account   string `json:"account"`
	Recurring bool   `json:"recurring"`
}

var webhookRules = []webhook.Rule{
	{
		Id:     WallosWebhookID,
		Secret: true,
		Handler: func(ctx types.Context, data []byte) types.MsgPayload {
			if ctx.Method != http.MethodPost {
				return types.TextMsg{Text: "Invalid request method. Use POST."}
			}

			var payload wallosWebhookData
			if err := json.Unmarshal(data, &payload); err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to parse webhook data: %v", err)}
			}

			msg := "*New Transaction Received*\n\n"

			if payload.Event != "" {
				msg += fmt.Sprintf("• *Event:* %s\n", payload.Event)
			}
			if payload.Amount != "" {
				amount := payload.Amount
				if payload.Currency != "" {
					amount = fmt.Sprintf("%s %s", payload.Currency, payload.Amount)
				}
				msg += fmt.Sprintf("*Amount:* %s\n", amount)
			}
			if payload.Category != "" {
				msg += fmt.Sprintf("• *Category:* %s\n", payload.Category)
			}
			if payload.Payee != "" {
				msg += fmt.Sprintf("• *Payee:* %s\n", payload.Payee)
			}
			if payload.Account != "" {
				msg += fmt.Sprintf("• *Account:* %s\n", payload.Account)
			}
			if payload.Date != "" {
				msg += fmt.Sprintf("• *Date:* %s\n", payload.Date)
			}
			if payload.Notes != "" {
				msg += fmt.Sprintf("• *Notes:* %s\n", payload.Notes)
			}
			if payload.Recurring {
				msg += "*Recurring:* Yes\n"
			}

			err := event.SendMessage(ctx, types.TextMsg{Text: msg})
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("Failed to send message: %v", err)}
			}

			return nil
		},
	},
}
