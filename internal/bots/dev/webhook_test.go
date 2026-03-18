package dev

import (
	"net/http"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookRules_Count(t *testing.T) {
	assert.Len(t, webhookRules, 2)
}

func TestWebhookRules_Constants(t *testing.T) {
	assert.Equal(t, "example", ExampleWebhookID)
	assert.Equal(t, "chat", ChatWebhookID)
}

func TestWebhookRules_IDs(t *testing.T) {
	ids := make(map[string]bool)
	for _, r := range webhookRules {
		ids[r.Id] = true
	}

	assert.True(t, ids[ExampleWebhookID])
	assert.True(t, ids[ChatWebhookID])
}

func TestWebhookRules_Secret(t *testing.T) {
	for _, r := range webhookRules {
		assert.True(t, r.Secret, "webhook %q should have Secret=true", r.Id)
	}
}

func TestWebhookRules_Handlers(t *testing.T) {
	for _, r := range webhookRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Id)
	}
}

func TestWebhookRules_Comprehensive(t *testing.T) {
	for _, r := range webhookRules {
		t.Run(r.Id, func(t *testing.T) {
			assert.NotEmpty(t, r.Id)
			assert.True(t, r.Secret)
			assert.NotNil(t, r.Handler)
		})
	}
}

func TestWebhookRules_ExampleHandler(t *testing.T) {
	var exampleRule *webhook.Rule
	for i := range webhookRules {
		if webhookRules[i].Id == ExampleWebhookID {
			exampleRule = &webhookRules[i]
			break
		}
	}
	require.NotNil(t, exampleRule)
	require.NotNil(t, exampleRule.Handler)

	tests := []struct {
		name         string
		method       string
		data         []byte
		wantMsgType  string
		wantContains string
	}{
		{
			name:         "POST request",
			method:       http.MethodPost,
			data:         []byte(`{"test":"data"}`),
			wantMsgType:  "TextMsg",
			wantContains: "POST",
		},
		{
			name:         "GET request",
			method:       http.MethodGet,
			data:         []byte(`test data`),
			wantMsgType:  "TextMsg",
			wantContains: "GET",
		},
		{
			name:         "empty data",
			method:       http.MethodPost,
			data:         []byte(``),
			wantMsgType:  "TextMsg",
			wantContains: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.Context{
				Platform: "test",
				Topic:    "test",
				AsUser:   types.Uid("test_user"),
				Method:   tt.method,
			}

			payload := exampleRule.Handler(ctx, tt.data)
			require.NotNil(t, payload)

			assert.Equal(t, tt.wantMsgType, types.TypeOf(payload))

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Contains(t, msg.Text, tt.wantContains)
		})
	}
}

func TestWebhookRules_ChatHandler(t *testing.T) {
	var chatRule *webhook.Rule
	for i := range webhookRules {
		if webhookRules[i].Id == ChatWebhookID {
			chatRule = &webhookRules[i]
			break
		}
	}
	require.NotNil(t, chatRule)
	require.NotNil(t, chatRule.Handler)

	ctx := types.Context{
		Platform: "test",
		Topic:    "test",
		AsUser:   types.Uid("test_user"),
		Method:   http.MethodGet,
	}

	payload := chatRule.Handler(ctx, []byte(`{}`))
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "error method")
}

func TestWebhookRules_ChatHandler_AgentDisabled(t *testing.T) {
	var chatRule *webhook.Rule
	for i := range webhookRules {
		if webhookRules[i].Id == ChatWebhookID {
			chatRule = &webhookRules[i]
			break
		}
	}
	require.NotNil(t, chatRule)
	require.NotNil(t, chatRule.Handler)

	tests := []struct {
		name         string
		method       string
		data         []byte
		wantContains string
	}{
		{
			name:         "wrong method",
			method:       http.MethodGet,
			data:         []byte(`{"text":"hello"}`),
			wantContains: "error method",
		},
		{
			name:         "agent disabled",
			method:       http.MethodPost,
			data:         []byte(`{"text":"hello","ip":"127.0.0.1"}`),
			wantContains: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.Context{
				Platform: "test",
				Topic:    "test",
				AsUser:   types.Uid("test_user"),
				Method:   tt.method,
			}

			payload := chatRule.Handler(ctx, tt.data)
			require.NotNil(t, payload)

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Contains(t, msg.Text, tt.wantContains)
		})
	}
}

func TestWebhookRuleset_ProcessRule(t *testing.T) {
	rs := webhook.Ruleset(webhookRules)
	ctx := types.Context{
		Platform:      "test",
		Topic:         "test",
		AsUser:        types.Uid("test_user"),
		WebhookRuleId: ExampleWebhookID,
		Method:        http.MethodPost,
	}

	payload, err := rs.ProcessRule(ctx, []byte(`test data`))
	require.NoError(t, err)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "POST")
}

func TestWebhookRuleset_ProcessRule_NotFound(t *testing.T) {
	rs := webhook.Ruleset(webhookRules)
	ctx := types.Context{
		Platform:      "test",
		Topic:         "test",
		AsUser:        types.Uid("test_user"),
		WebhookRuleId: "nonexistent",
		Method:        http.MethodPost,
	}

	payload, err := rs.ProcessRule(ctx, []byte(`test data`))
	require.NoError(t, err)
	assert.Nil(t, payload)
}
