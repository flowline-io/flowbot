package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHasScope(t *testing.T) {
	require.True(t, HasScope([]string{ScopeAdmin}, ScopeHubAppsRead))
	require.True(t, HasScope([]string{ScopeHubAppsRead}, ScopeHubAppsRead))
	require.False(t, HasScope([]string{ScopeHubAppsStatus}, ScopeHubAppsRead))
}

func TestExtractBearerToken(t *testing.T) {
	require.Equal(t, "token", ExtractBearerToken("Bearer token"))
	require.Equal(t, "token", ExtractBearerToken("bearer token"))
	require.Equal(t, "token", ExtractBearerToken("token"))
}

func TestWebhookSignature(t *testing.T) {
	now := time.Unix(1700000000, 0)
	body := []byte(`{"url":"https://example.com"}`)
	signature := SignWebhook("secret", "post", "/webhook/bookmark/create", now, body)

	require.True(t, VerifyWebhookSignature("secret", "POST", "/webhook/bookmark/create", now, body, signature, now, time.Minute))
	require.False(t, VerifyWebhookSignature("secret", "POST", "/webhook/bookmark/create", now.Add(-2*time.Minute), body, signature, now, time.Minute))
}
