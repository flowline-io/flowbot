package dev

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
)

func TestWebhookRules_Metadata(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 1 webhook rule",
			test: func(t *testing.T) {
				assert.Len(t, webhookRules, 1)
			},
		},
		{
			name: "should have correct constant",
			test: func(t *testing.T) {
				assert.Equal(t, "example", ExampleWebhookID)
			},
		},
		{
			name: "should contain ExampleWebhookID",
			test: func(t *testing.T) {
				ids := make(map[string]bool)
				for _, r := range webhookRules {
					ids[r.Id] = true
				}

				assert.True(t, ids[ExampleWebhookID])
			},
		},
		{
			name: "all webhooks should have Secret=true",
			test: func(t *testing.T) {
				for _, r := range webhookRules {
					assert.True(t, r.Secret, "webhook %q should have Secret=true", r.Id)
				}
			},
		},
		{
			name: "all webhook handlers should be non-nil",
			test: func(t *testing.T) {
				for _, r := range webhookRules {
					assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Id)
				}
			},
		},
		{
			name: "all webhooks should have non-empty ID, Secret=true, and non-nil handler",
			test: func(t *testing.T) {
				for _, r := range webhookRules {
					t.Run(r.Id, func(t *testing.T) {
						assert.NotEmpty(t, r.Id)
						assert.True(t, r.Secret)
						assert.NotNil(t, r.Handler)
					})
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestWebhookRules_ExampleHandler(t *testing.T) {
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
			var exampleRule *webhook.Rule
			for i := range webhookRules {
				if webhookRules[i].Id == ExampleWebhookID {
					exampleRule = &webhookRules[i]
					break
				}
			}
			require.NotNil(t, exampleRule)
			require.NotNil(t, exampleRule.Handler)

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

func TestWebhookRuleset_ProcessRule(t *testing.T) {
	tests := []struct {
		name        string
		ruleID      string
		wantErr     bool
		wantNil     bool
		wantContain string
	}{
		{
			name:        "existing rule returns payload",
			ruleID:      ExampleWebhookID,
			wantErr:     false,
			wantNil:     false,
			wantContain: "POST",
		},
		{
			name:        "nonexistent rule returns nil payload",
			ruleID:      "nonexistent",
			wantErr:     false,
			wantNil:     true,
			wantContain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := webhook.Ruleset(webhookRules)
			ctx := types.Context{
				Platform:      "test",
				Topic:         "test",
				AsUser:        types.Uid("test_user"),
				WebhookRuleId: tt.ruleID,
				Method:        http.MethodPost,
			}

			payload, err := rs.ProcessRule(ctx, []byte(`test data`))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, payload)
			} else {
				require.NotNil(t, payload)
				msg, ok := payload.(types.TextMsg)
				require.True(t, ok)
				if tt.wantContain != "" {
					assert.Contains(t, msg.Text, tt.wantContain)
				}
			}
		})
	}
}
