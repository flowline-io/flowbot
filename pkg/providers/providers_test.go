package providers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestRedirectURI(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Run("multiple config calls", func(t *testing.T) {
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
	})
}

func TestOAuthProviderInterface(t *testing.T) {
	t.Parallel()
	t.Run("interface compile-time check", func(t *testing.T) {
		t.Parallel()
		var _ OAuthProvider = (*mockOAuthProvider)(nil)
	})
}

type mockOAuthProvider struct{}

func (*mockOAuthProvider) GetAuthorizeURL(state string) string {
	return "https://example.com/auth?state=" + state
}

func (*mockOAuthProvider) GetAccessToken(_ fiber.Ctx) (*OAuthToken, error) {
	return &OAuthToken{AccessToken: "test", Type: "mock"}, nil
}

type fakeOAuthTokenStore struct {
	token *OAuthToken
	err   error
	sets  int
}

func (f *fakeOAuthTokenStore) Get(_ context.Context, _ types.Uid, _, _ string) (*OAuthToken, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.token == nil {
		return nil, types.ErrNotFound
	}
	cp := *f.token
	if f.token.ExpiresAt != nil {
		t := *f.token.ExpiresAt
		cp.ExpiresAt = &t
	}
	return &cp, nil
}

func (f *fakeOAuthTokenStore) Set(_ context.Context, _ types.Uid, _ string, token *OAuthToken) error {
	f.sets++
	cp := *token
	if token.ExpiresAt != nil {
		t := *token.ExpiresAt
		cp.ExpiresAt = &t
	}
	f.token = &cp
	return nil
}

type refreshOAuthProvider struct {
	mockOAuthProvider
	refreshToken string
	err          error
}

func (r *refreshOAuthProvider) RefreshAccessToken(_ context.Context, refreshToken string) (*OAuthToken, error) {
	if r.err != nil {
		return nil, r.err
	}
	exp := time.Now().Add(time.Hour)
	return &OAuthToken{
		AccessToken:  "refreshed",
		RefreshToken: refreshToken,
		ExpiresAt:    &exp,
		Type:         "refreshable",
		Name:         "refreshed-name",
	}, nil
}

func TestGetOrRefreshToken(t *testing.T) {
	uid := types.Uid("user1")
	const topic = "topic"
	const providerName = "refreshable"

	t.Cleanup(func() {
		SetOAuthTokenStore(nil)
		UnregisterOAuthProvider(providerName)
	})

	t.Run("store not configured", func(t *testing.T) {
		SetOAuthTokenStore(nil)
		_, err := GetOrRefreshToken(context.Background(), uid, topic, providerName)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured")
	})

	t.Run("returns unexpired token", func(t *testing.T) {
		exp := time.Now().Add(time.Hour)
		fake := &fakeOAuthTokenStore{token: &OAuthToken{
			Name: "n", Type: providerName, AccessToken: "tok", ExpiresAt: &exp,
		}}
		SetOAuthTokenStore(fake)
		got, err := GetOrRefreshToken(context.Background(), uid, topic, providerName)
		require.NoError(t, err)
		assert.Equal(t, "tok", got.AccessToken)
		assert.Equal(t, 0, fake.sets)
	})

	t.Run("refreshes expired token", func(t *testing.T) {
		exp := time.Now().Add(-time.Minute)
		fake := &fakeOAuthTokenStore{token: &OAuthToken{
			Name: "n", Type: providerName, AccessToken: "old", RefreshToken: "rt", ExpiresAt: &exp,
		}}
		SetOAuthTokenStore(fake)
		RegisterOAuthProvider(providerName, func() OAuthProvider {
			return &refreshOAuthProvider{}
		})
		got, err := GetOrRefreshToken(context.Background(), uid, topic, providerName)
		require.NoError(t, err)
		assert.Equal(t, "refreshed", got.AccessToken)
		assert.Equal(t, 1, fake.sets)
		assert.Equal(t, "refreshed", fake.token.AccessToken)
	})

	t.Run("expired without refresher", func(t *testing.T) {
		exp := time.Now().Add(-time.Minute)
		fake := &fakeOAuthTokenStore{token: &OAuthToken{
			Name: "n", Type: "norefresh", AccessToken: "old", RefreshToken: "rt", ExpiresAt: &exp,
		}}
		SetOAuthTokenStore(fake)
		RegisterOAuthProvider("norefresh", func() OAuthProvider {
			return &mockOAuthProvider{}
		})
		t.Cleanup(func() { UnregisterOAuthProvider("norefresh") })
		_, err := GetOrRefreshToken(context.Background(), uid, topic, "norefresh")
		require.Error(t, err)
		assert.ErrorIs(t, err, types.ErrForbidden)
	})
}

