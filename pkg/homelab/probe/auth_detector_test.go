package probe

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthDetector_BearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	detector := &AuthDetector{}
	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	auth := detector.Detect(resp)
	require.NotNil(t, auth)
	assert.Equal(t, "oauth2", string(auth.Type))
	assert.Equal(t, "Authorization", auth.Header)
	assert.Equal(t, "Bearer", auth.Prefix)
}

func TestAuthDetector_BasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	detector := &AuthDetector{}
	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	auth := detector.Detect(resp)
	require.NotNil(t, auth)
	assert.Equal(t, "basic", string(auth.Type))
	assert.Equal(t, "Authorization", auth.Header)
	assert.Equal(t, "Basic", auth.Prefix)
}

func TestAuthDetector_NoAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	detector := &AuthDetector{}
	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	auth := detector.Detect(resp)
	require.NotNil(t, auth)
	assert.Equal(t, "none", string(auth.Type))
}

func TestAuthDetector_ForbiddenNoWWWAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	detector := &AuthDetector{}
	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	auth := detector.Detect(resp)
	require.NotNil(t, auth)
	// 403 without WWW-Authenticate suggests API key auth.
	assert.Equal(t, "api_token", string(auth.Type))
}

func TestAuthDetector_OIDCHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="oidc"`)
		w.Header().Set("X-OIDC-Issuer", "https://issuer.example.com")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	detector := &AuthDetector{}
	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	auth := detector.Detect(resp)
	require.NotNil(t, auth)
	assert.Equal(t, "oidc", string(auth.Type))
}

func TestAuthDetector_UnknownAuthScheme(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Digest realm="protected"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	detector := &AuthDetector{}
	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	auth := detector.Detect(resp)
	require.NotNil(t, auth)
	assert.Equal(t, "unknown", string(auth.Type))
}

func TestAuthDetector_NilResponse(t *testing.T) {
	detector := &AuthDetector{}
	auth := detector.Detect(nil)
	assert.Nil(t, auth)
}
