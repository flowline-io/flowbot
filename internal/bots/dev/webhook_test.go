package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
