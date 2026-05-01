package github

import (
	"net/http"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWebhookHandler_Ping(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodPost,
		Headers: map[string][]string{
			"X-Github-Event": {"ping"},
		},
	}
	result := handler(ctx, []byte(`{}`))

	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "pong", msg.Text)
}

func TestWebhookHandler_WrongMethod(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodGet,
		Headers: map[string][]string{
			"X-Github-Event": {"package"},
		},
	}
	result := handler(ctx, []byte(`{}`))

	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "error method", msg.Text)
}

func TestWebhookHandler_MissingEventHeader(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method:  http.MethodPost,
		Headers: map[string][]string{},
	}
	result := handler(ctx, []byte(`{}`))

	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "error header", msg.Text)
}

func TestWebhookHandler_EmptyEventHeader(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodPost,
		Headers: map[string][]string{
			"X-Github-Event": {},
		},
	}
	result := handler(ctx, []byte(`{}`))

	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "error event", msg.Text)
}

func TestWebhookHandler_UnsupportedEvent(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodPost,
		Headers: map[string][]string{
			"X-Github-Event": {"push"},
		},
	}
	result := handler(ctx, []byte(`{}`))

	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "not supported", msg.Text)
}

func TestWebhookHandler_PackageNotLatest(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodPost,
		Headers: map[string][]string{
			"X-Github-Event": {"package"},
		},
	}
	data := `{
		"action": "published",
		"package": {
			"id": 12345,
			"name": "test-package",
			"namespace": "testuser",
			"description": "A test package",
			"ecosystem": "docker",
			"package_type": "container",
			"html_url": "https://github.com/testuser/test-repo/pkgs/container/test-package",
			"package_version": {
				"id": 67890,
				"version": "1.0.0",
				"name": "test-package:1.0.0",
				"description": "Version 1.0.0",
				"container_metadata": {
					"tag": {
						"name": "1.0.0",
						"digest": "sha256:abc123"
					}
				}
			}
		}
	}`

	result := handler(ctx, []byte(data))
	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "not latest", msg.Text)
}

func TestWebhookHandler_PackageNotPublished(t *testing.T) {
	handler := webhookRules[0].Handler
	ctx := types.Context{
		Method: http.MethodPost,
		Headers: map[string][]string{
			"X-Github-Event": {"package"},
		},
	}
	data := `{
		"action": "updated",
		"package": {
			"id": 12345,
			"name": "test-package",
			"namespace": "testuser",
			"description": "A test package",
			"ecosystem": "docker",
			"package_type": "container",
			"html_url": "https://github.com/testuser/test-repo/pkgs/container/test-package",
			"package_version": {
				"id": 67890,
				"version": "1.0.0",
				"name": "test-package:1.0.0",
				"description": "Version 1.0.0",
				"container_metadata": {
					"tag": {
						"name": "latest",
						"digest": "sha256:abc123"
					}
				}
			}
		}
	}`

	result := handler(ctx, []byte(data))
	msg, ok := result.(types.TextMsg)
	require.True(t, ok)
	assert.Equal(t, "not published", msg.Text)
}

func TestWebhookRule_ImplementsInterface(t *testing.T) {
	var _ webhook.Rule = webhookRules[0]
}
