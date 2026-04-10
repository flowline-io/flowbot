package finance

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookConstants(t *testing.T) {
	assert.Equal(t, "wallos", WallosWebhookID)
}

func TestWebhookRules_Count(t *testing.T) {
	assert.Len(t, webhookRules, 1)
}

func TestWebhookRules_ID(t *testing.T) {
	assert.Equal(t, WallosWebhookID, webhookRules[0].Id)
}

func TestWebhookRules_Secret(t *testing.T) {
	assert.True(t, webhookRules[0].Secret)
}

func TestWebhookRules_Handler(t *testing.T) {
	assert.NotNil(t, webhookRules[0].Handler)
}

func TestWebhookHandler_WrongMethod(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodGet,
	}
	result := handler(ctx, nil)

	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "Invalid request method. Use POST.", msg.Text)
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodPost,
	}
	result := handler(ctx, []byte(`{invalid json`))

	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "Failed to parse webhook data")
}

func TestWallosWebhookData_Unmarshal(t *testing.T) {
	data := `{
		"event": "subscription_created",
		"amount": "9.99",
		"currency": "USD",
		"category": "entertainment",
		"payee": "Netflix",
		"notes": "Monthly subscription",
		"date": "2025-01-01",
		"account": "main",
		"recurring": true
	}`

	var payload wallosWebhookData
	err := json.Unmarshal([]byte(data), &payload)
	require.NoError(t, err)

	assert.Equal(t, "subscription_created", payload.Event)
	assert.Equal(t, "9.99", payload.Amount)
	assert.Equal(t, "USD", payload.Currency)
	assert.Equal(t, "entertainment", payload.Category)
	assert.Equal(t, "Netflix", payload.Payee)
	assert.Equal(t, "Monthly subscription", payload.Notes)
	assert.Equal(t, "2025-01-01", payload.Date)
	assert.Equal(t, "main", payload.Account)
	assert.True(t, payload.Recurring)
}

func TestWallosWebhookData_Fields(t *testing.T) {
	d := wallosWebhookData{
		Event:    "test",
		Amount:   "5.00",
		Currency: "EUR",
		Category: "food",
		Payee:    "Store",
		Notes:    "Groceries",
		Date:     "2025-01-15",
		Account:  "checking",
	}
	assert.Equal(t, "test", d.Event)
	assert.Equal(t, "5.00", d.Amount)
	assert.Equal(t, "EUR", d.Currency)
	assert.Equal(t, "food", d.Category)
	assert.Equal(t, "Store", d.Payee)
}

func TestWebhookRule_ImplementsInterface(t *testing.T) {
	var _ webhook.Rule = webhookRules[0]
}
