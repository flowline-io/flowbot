package reader

import (
	"net/http"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookConstants(t *testing.T) {
	assert.Equal(t, "miniflux", MinifluxWebhookID)
}

func TestWebhookRules_Count(t *testing.T) {
	assert.Len(t, webhookRules, 1)
}

func TestWebhookRules_ID(t *testing.T) {
	assert.Equal(t, MinifluxWebhookID, webhookRules[0].Id)
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
	assert.Equal(t, "error method", msg.Text)
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodPost,
	}
	result := handler(ctx, []byte(`{invalid json`))

	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "error event response", msg.Text)
}

func TestWebhookHandler_NewEntriesEvent(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodPost,
	}
	data := `{
		"event_type": "new_entries",
		"entry": {
			"id": 123,
			"title": "Test Entry",
			"url": "https://example.com/article"
		}
	}`

	result := handler(ctx, []byte(data))
	assert.Nil(t, result)
}

func TestWebhookRule_ImplementsInterface(t *testing.T) {
	var _ webhook.Rule = webhookRules[0]
}
