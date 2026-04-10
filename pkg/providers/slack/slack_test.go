package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlack_GetAuthorizeURL(t *testing.T) {
	tests := []struct {
		name        string
		clientID    string
		redirectURI string
		state       string
		wantParts   []string
	}{
		{
			name:        "basic URL generation",
			clientID:    "123.456",
			redirectURI: "https://example.com/oauth/slack/callback",
			state:       "",
			wantParts: []string{
				"https://slack.com/oauth/v2/authorize",
				"client_id=123.456",
				"user_scope=identity.basic%2Cidentity.avatar",
				"redirect_uri=https%3A%2F%2Fexample.com%2Foauth%2Fslack%2Fcallback",
			},
		},
		{
			name:        "URL with state",
			clientID:    "789.012",
			redirectURI: "https://app.example.com/callback",
			state:       "csrf-token-123",
			wantParts: []string{
				"client_id=789.012",
				"state=csrf-token-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slack := NewSlack(tt.clientID, "secret", tt.redirectURI, "")
			if tt.state != "" {
				slack.SetState(tt.state)
			}

			got := slack.GetAuthorizeURL()
			for _, part := range tt.wantParts {
				assert.Contains(t, got, part)
			}
		})
	}
}

func TestSlack_completeAuth(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		response   OAuthV2AccessResponse
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful token exchange",
			code: "valid-code",
			response: OAuthV2AccessResponse{
				OK:    true,
				Error: "",
				AuthedUser: struct {
					ID          string `json:"id"`
					Scope       string `json:"scope"`
					AccessToken string `json:"access_token"`
					TokenType   string `json:"token_type"`
				}{
					ID:          "U123456",
					Scope:       "identity.basic,identity.avatar",
					AccessToken: "xoxp-test-token",
					TokenType:   "user",
				},
				Team: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					ID:   "T123",
					Name: "My Team",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "access denied",
			code: "invalid-code",
			response: OAuthV2AccessResponse{
				OK:    false,
				Error: "invalid_code",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "server error",
			code:       "any-code",
			response:   OAuthV2AccessResponse{},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/oauth.v2.access", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

				body, _ := io.ReadAll(r.Body)
				values, _ := url.ParseQuery(string(body))
				assert.Equal(t, "test-client-id", values.Get("client_id"))
				assert.Equal(t, "test-secret", values.Get("client_secret"))
				assert.Equal(t, tt.code, values.Get("code"))

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			slack := NewSlack("test-client-id", "test-secret", "https://example.com/callback", "")
			slack.c.SetBaseURL(server.URL)

			result, err := slack.completeAuth(tt.code)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "xoxp-test-token", result.AuthedUser.AccessToken)
				assert.Equal(t, "U123456", result.AuthedUser.ID)
				assert.Equal(t, "My Team", result.Team.Name)
			}
		})
	}
}

func TestSlack_GetIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users.identity", r.URL.Path)
		assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "))

		response := IdentityResponse{
			OK:    true,
			Error: "",
			User: struct {
				Name    string `json:"name"`
				ID      string `json:"id"`
				Image48 string `json:"image_48"`
			}{
				Name:    "Test User",
				ID:      "U123456",
				Image48: "https://example.com/avatar.png",
			},
			Team: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   "T123",
				Name: "Test Team",
			},
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer server.Close()

	slack := NewSlack("client-id", "secret", "https://example.com/callback", "xoxp-access-token")
	slack.c.SetBaseURL(server.URL)

	result, err := slack.GetIdentity()

	require.NoError(t, err)
	assert.Equal(t, "Test User", result.User.Name)
	assert.Equal(t, "U123456", result.User.ID)
	assert.Equal(t, "Test Team", result.Team.Name)
}

func TestSlack_GetIdentity_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := IdentityResponse{
			OK:    false,
			Error: "account_inactive",
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer server.Close()

	slack := NewSlack("client-id", "secret", "https://example.com/callback", "invalid-token")
	slack.c.SetBaseURL(server.URL)

	_, err := slack.GetIdentity()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account_inactive")
}

func TestNewSlack(t *testing.T) {
	slack := NewSlack("123.456", "secret", "https://example.com/callback", "xoxp-token")
	assert.NotNil(t, slack)
	assert.NotNil(t, slack.c)
	assert.Equal(t, "123.456", slack.clientId)
	assert.Equal(t, "secret", slack.clientSecret)
	assert.Equal(t, "https://example.com/callback", slack.redirectURI)
	assert.Equal(t, "xoxp-token", slack.accessToken)
}
