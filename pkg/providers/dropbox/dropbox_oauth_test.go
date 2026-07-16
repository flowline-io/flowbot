package dropbox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantID  string
	}{
		{name: "empty config", configs: json.RawMessage(`{}`), wantID: ""},
		{name: "reads client credentials", configs: json.RawMessage(`{"dropbox":{"key":"cid","secret":"sec"}}`), wantID: "cid"},
		{name: "key only", configs: json.RawMessage(`{"dropbox":{"key":"cid"}}`), wantID: "cid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			d := GetClient()
			require.NotNil(t, d)
			assert.Equal(t, tt.wantID, d.clientId)
		})
	}
}

func TestDropbox_completeAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		code       string
		statusCode int
		body       string
		wantErr    bool
		wantToken  string
	}{
		{
			name:       "successful token exchange",
			code:       "auth-code",
			statusCode: http.StatusOK,
			body:       `{"access_token":"at-123","token_type":"bearer","expires_in":3600,"refresh_token":"rt-456","scope":"files.content.read","uid":"u1","account_id":"a1"}`,
			wantToken:  "at-123",
		},
		{
			name:       "invalid code",
			code:       "bad",
			statusCode: http.StatusBadRequest,
			body:       `{"error":"invalid_grant"}`,
			wantErr:    true,
		},
		{
			name:       "malformed json",
			code:       "auth-code",
			statusCode: http.StatusOK,
			body:       `{`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/oauth2/token", r.URL.Path)
				user, pass, ok := r.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "client-id", user)
				assert.Equal(t, "client-secret", pass)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			d := NewDropbox("client-id", "client-secret", "https://example.com/callback", "")
			d.c.SetBaseURL(srv.URL)

			token, err := d.completeAuth(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantToken, token.AccessToken)
			assert.Equal(t, tt.wantToken, d.accessToken)
		})
	}
}

func TestDropbox_RefreshAccessToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantToken  string
	}{
		{
			name:       "refreshes token",
			statusCode: http.StatusOK,
			body:       `{"access_token":"new-at","token_type":"bearer","expires_in":7200,"refresh_token":"new-rt","scope":"files.content.read"}`,
			wantToken:  "new-at",
		},
		{
			name:       "invalid refresh token",
			statusCode: http.StatusBadRequest,
			body:       `{"error":"invalid_grant"}`,
			wantErr:    true,
		},
		{
			name:       "parse error",
			statusCode: http.StatusOK,
			body:       `not-json`,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/oauth2/token", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			d := NewDropbox("client-id", "client-secret", "", "")
			d.c.SetBaseURL(srv.URL)

			token, err := d.RefreshAccessToken(context.Background(), "refresh-token")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantToken, token.AccessToken)
			assert.Equal(t, "dropbox", token.Name)
		})
	}
}

func TestDropbox_Upload_ReadError(t *testing.T) {
	t.Parallel()
	d := NewDropbox("id", "secret", "", "token")
	err := d.Upload("/x", errReader{})
	assert.Error(t, err)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestRegister(t *testing.T) {
	t.Parallel()
	providers.UnregisterOAuthProvider(ID)
	Register()
	p, err := providers.GetOAuthProvider(ID)
	require.NoError(t, err)
	require.NotNil(t, p)
	providers.UnregisterOAuthProvider(ID)
}
