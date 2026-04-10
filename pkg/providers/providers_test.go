package providers

import (
	"encoding/json"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectURI(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		flag     string
		want     string
	}{
		{
			name:     "github oauth redirect",
			provider: "github",
			flag:     "callback",
			want:     "/oauth/github/callback",
		},
		{
			name:     "empty flag",
			provider: "test",
			flag:     "",
			want:     "/oauth/test/",
		},
		{
			name:     "provider with hyphen",
			provider: "my-provider",
			flag:     "auth",
			want:     "/oauth/my-provider/auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedirectURI(tt.provider, tt.flag)
			assert.Contains(t, got, tt.want)
		})
	}
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name      string
		configs   json.RawMessage
		nameKey   string
		key       string
		wantErr   bool
		wantValue string
	}{
		{
			name:      "empty configs",
			configs:   json.RawMessage{},
			nameKey:   "test",
			key:       "key",
			wantErr:   true,
			wantValue: "",
		},
		{
			name:      "valid config",
			configs:   json.RawMessage(`{"github":{"client_id":"test123"}}`),
			nameKey:   "github",
			key:       "client_id",
			wantErr:   false,
			wantValue: "test123",
		},
		{
			name:      "missing key",
			configs:   json.RawMessage(`{"github":{"client_id":"test123"}}`),
			nameKey:   "github",
			key:       "missing_key",
			wantErr:   false,
			wantValue: "",
		},
		{
			name:      "nested config",
			configs:   json.RawMessage(`{"provider":{"nested":{"key":"value"}}}`),
			nameKey:   "provider",
			key:       "nested.key",
			wantErr:   false,
			wantValue: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Configs = tt.configs
			result, err := GetConfig(tt.nameKey, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantValue, result.String())
			}
		})
	}
}

func TestGetConfig_MultipleCalls(t *testing.T) {
	Configs = json.RawMessage(`{
		"github": {
			"client_id": "id123",
			"client_secret": "secret456"
		},
		"slack": {
			"token": "xoxb-test"
		}
	}`)

	githubID, err := GetConfig("github", "client_id")
	require.NoError(t, err)
	assert.Equal(t, "id123", githubID.String())

	githubSecret, err := GetConfig("github", "client_secret")
	require.NoError(t, err)
	assert.Equal(t, "secret456", githubSecret.String())

	slackToken, err := GetConfig("slack", "token")
	require.NoError(t, err)
	assert.Equal(t, "xoxb-test", slackToken.String())
}

func TestOAuthProviderInterface(t *testing.T) {
	// Test that OAuthProvider interface is defined correctly
	// This is a compile-time check
	var _ OAuthProvider = (*mockOAuthProvider)(nil)
}

type mockOAuthProvider struct{}

func (m *mockOAuthProvider) GetAuthorizeURL() string {
	return "https://example.com/auth"
}

func (m *mockOAuthProvider) GetAccessToken(ctx fiber.Ctx) (types.KV, error) {
	return types.KV{"token": "test"}, nil
}
