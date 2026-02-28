package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookConstants(t *testing.T) {
	assert.Equal(t, "package", PackageWebhookID)
}

func TestWebhookRules_Count(t *testing.T) {
	assert.Len(t, webhookRules, 1)
}

func TestWebhookRules_ID(t *testing.T) {
	assert.Equal(t, PackageWebhookID, webhookRules[0].Id)
}

func TestWebhookRules_Secret(t *testing.T) {
	assert.True(t, webhookRules[0].Secret)
}

func TestWebhookRules_Handler(t *testing.T) {
	assert.NotNil(t, webhookRules[0].Handler)
}
