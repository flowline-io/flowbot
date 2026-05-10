package dropbox

import (
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenResponse_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal token response", func(t *testing.T) {
		t.Parallel()
		data := `{
			"access_token": "test_token_123",
			"token_type": "bearer",
			"uid": "user123",
			"account_id": "dbid:test_account",
			"scope": "files.content.write files.content.read"
		}`

		var token TokenResponse
		err := sonic.Unmarshal([]byte(data), &token)
		require.NoError(t, err)
		assert.Equal(t, "test_token_123", token.AccessToken)
		assert.Equal(t, "bearer", token.TokenType)
		assert.Equal(t, "user123", token.UID)
		assert.Equal(t, "dbid:test_account", token.AccountID)
		assert.Equal(t, "files.content.write files.content.read", token.Scope)
	})
}

func TestConstants(t *testing.T) {
	t.Parallel()
	t.Run("dropbox constants", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "dropbox", ID)
		assert.Equal(t, "key", ClientIdKey)
		assert.Equal(t, "secret", ClientSecretKey)
	})
}

func TestDropbox_Constructor(t *testing.T) {
	t.Parallel()
	t.Run("constructor sets fields", func(t *testing.T) {
		t.Parallel()
		clientId := "test_client_id"
		clientSecret := "test_client_secret"
		redirectURI := "https://example.com/callback"
		accessToken := "test_access_token"

		dropbox := NewDropbox(clientId, clientSecret, redirectURI, accessToken)

		assert.NotNil(t, dropbox)
		assert.Equal(t, clientId, dropbox.clientId)
		assert.Equal(t, clientSecret, dropbox.clientSecret)
		assert.Equal(t, redirectURI, dropbox.redirectURI)
		assert.Equal(t, accessToken, dropbox.accessToken)
	})
}

func TestDropbox_GetAuthorizeURL(t *testing.T) {
	t.Parallel()
	t.Run("authorize URL generation", func(t *testing.T) {
		t.Parallel()
		dropbox := NewDropbox("client_id", "secret", "https://example.com/callback", "")
		url := dropbox.GetAuthorizeURL()

		assert.Contains(t, url, "https://www.dropbox.com/oauth2/authorize")
		assert.Contains(t, url, "client_id=client_id")
		assert.Contains(t, url, "response_type=code")
		assert.Contains(t, url, "redirect_uri=https://example.com/callback")
	})
}

func TestDropbox_Redirect(t *testing.T) {
	t.Parallel()
	t.Run("redirect URL generation", func(t *testing.T) {
		t.Parallel()
		dropbox := NewDropbox("client_id", "secret", "https://example.com/callback", "")
		url, err := dropbox.Redirect(nil)

		require.NoError(t, err)
		assert.Contains(t, url, "dropbox.com/oauth2/authorize")
	})
}
