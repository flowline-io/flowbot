package reader

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
)

func TestWebhookConstants(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "miniflux webhook id constant",
			fn: func(t *testing.T) {
				assert.Equal(t, "miniflux", MinifluxWebhookID)
			},
		},
		{
			name: "one webhook rule",
			fn: func(t *testing.T) {
				assert.Len(t, webhookRules, 1)
			},
		},
		{
			name: "webhook rule id matches constant",
			fn: func(t *testing.T) {
				assert.Equal(t, MinifluxWebhookID, webhookRules[0].Id)
			},
		},
		{
			name: "webhook rule has secret",
			fn: func(t *testing.T) {
				assert.True(t, webhookRules[0].Secret)
			},
		},
		{
			name: "webhook rule handler not nil",
			fn: func(t *testing.T) {
				assert.NotNil(t, webhookRules[0].Handler)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}

func TestWebhookHandler(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		body     []byte
		wantNil  bool
		wantText string
	}{
		{
			name:     "wrong method returns error method text",
			method:   http.MethodGet,
			body:     nil,
			wantText: "error method",
		},
		{
			name:     "invalid json returns error event response text",
			method:   http.MethodPost,
			body:     []byte(`{invalid json`),
			wantText: "error event response",
		},
		{
			name:   "new entries event returns nil",
			method: http.MethodPost,
			body: []byte(`{
		"event_type": "new_entries",
		"entry": {
			"id": 123,
			"title": "Test Entry",
			"url": "https://example.com/article"
		}
	}`),
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := webhookRules[0].Handler
			ctx := types.Context{
				Method: tt.method,
			}
			result := handler(ctx, tt.body)

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				msg, ok := result.(types.TextMsg)
				require.True(t, ok)
				assert.Equal(t, tt.wantText, msg.Text)
			}
		})
	}
}

func TestWebhookRule_ImplementsInterface(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "webhook rule implements webhook.Rule interface"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ webhook.Rule = webhookRules[0]
		})
	}
}
