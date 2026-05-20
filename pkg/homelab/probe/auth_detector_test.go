package probe

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthDetector_BearerAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		wwwAuth      string
		statusCode   int
		wantType     string
		wantHeader   string
		wantPrefix   string
		extraHeaders map[string]string
	}{
		{
			name:       "detects Bearer OAuth2 from WWW-Authenticate",
			wwwAuth:    `Bearer realm="api"`,
			statusCode: http.StatusUnauthorized,
			wantType:   "oauth2",
			wantHeader: "Authorization",
			wantPrefix: "Bearer",
		},
		{
			name:       "detects oauth2 with Bearer and extra attributes",
			wwwAuth:    `Bearer realm="api", error="invalid_token", error_description="expired"`,
			statusCode: http.StatusUnauthorized,
			wantType:   "oauth2",
			wantHeader: "Authorization",
			wantPrefix: "Bearer",
		},
		{
			name:       "returns unknown for lowercase bearer prefix",
			wwwAuth:    `bearer realm="api"`,
			statusCode: http.StatusUnauthorized,
			wantType:   "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("WWW-Authenticate", tt.wwwAuth)
				for k, v := range tt.extraHeaders {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			detector := &AuthDetector{}
			resp, err := server.Client().Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close()

			auth := detector.Detect(resp)
			require.NotNil(t, auth)
			assert.Equal(t, tt.wantType, string(auth.Type))
			if tt.wantHeader != "" {
				assert.Equal(t, tt.wantHeader, auth.Header)
			}
			if tt.wantPrefix != "" {
				assert.Equal(t, tt.wantPrefix, auth.Prefix)
			}
		})
	}
}

func TestAuthDetector_BasicAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wwwAuth    string
		statusCode int
		wantType   string
		wantHeader string
		wantPrefix string
	}{
		{
			name:       "detects Basic auth from WWW-Authenticate",
			wwwAuth:    `Basic realm="restricted"`,
			statusCode: http.StatusUnauthorized,
			wantType:   "basic",
			wantHeader: "Authorization",
			wantPrefix: "Basic",
		},
		{
			name:       "detects Basic auth with charset parameter",
			wwwAuth:    `Basic realm="restricted", charset="UTF-8"`,
			statusCode: http.StatusUnauthorized,
			wantType:   "basic",
			wantHeader: "Authorization",
			wantPrefix: "Basic",
		},
		{
			name:       "detects api_token for 401 without WWW-Authenticate",
			wwwAuth:    "",
			statusCode: http.StatusUnauthorized,
			wantType:   "api_token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.wwwAuth != "" {
					w.Header().Set("WWW-Authenticate", tt.wwwAuth)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			detector := &AuthDetector{}
			resp, err := server.Client().Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close()

			auth := detector.Detect(resp)
			require.NotNil(t, auth)
			assert.Equal(t, tt.wantType, string(auth.Type))
			if tt.wantHeader != "" {
				assert.Equal(t, tt.wantHeader, auth.Header)
			}
			if tt.wantPrefix != "" {
				assert.Equal(t, tt.wantPrefix, auth.Prefix)
			}
		})
	}
}

func TestAuthDetector_NoAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		wantType   string
	}{
		{name: "detects no auth on 200 OK", statusCode: http.StatusOK, wantType: "none"},
		{name: "detects no auth on 204 No Content", statusCode: http.StatusNoContent, wantType: "none"},
		{name: "returns unknown for 500 server error", statusCode: http.StatusInternalServerError, wantType: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte("ok"))
			}))
			defer server.Close()

			detector := &AuthDetector{}
			resp, err := server.Client().Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close()

			auth := detector.Detect(resp)
			require.NotNil(t, auth)
			assert.Equal(t, tt.wantType, string(auth.Type))
		})
	}
}

func TestAuthDetector_ForbiddenNoWWWAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wwwAuth    string
		statusCode int
		wantType   string
	}{
		{
			name:       "detects API key auth on 403 without WWW-Authenticate",
			statusCode: http.StatusForbidden,
			wantType:   "api_token",
		},
		{
			name:       "detects API key auth on 401 without WWW-Authenticate",
			statusCode: http.StatusUnauthorized,
			wantType:   "api_token",
		},
		{
			name:       "returns unknown for 403 with unrecognized WWW-Authenticate",
			wwwAuth:    `Digest realm="protected"`,
			statusCode: http.StatusForbidden,
			wantType:   "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.wwwAuth != "" {
					w.Header().Set("WWW-Authenticate", tt.wwwAuth)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			detector := &AuthDetector{}
			resp, err := server.Client().Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close()

			auth := detector.Detect(resp)
			require.NotNil(t, auth)
			assert.Equal(t, tt.wantType, string(auth.Type))
		})
	}
}

func TestAuthDetector_OIDCHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		wwwAuth      string
		extraHeaders map[string]string
		wantType     string
		wantHeader   string
		wantPrefix   string
	}{
		{
			name:    "detects OIDC when X-OIDC-Issuer header present",
			wwwAuth: `Bearer realm="oidc"`,
			extraHeaders: map[string]string{
				"X-OIDC-Issuer": "https://issuer.example.com",
			},
			wantType:   "oidc",
			wantHeader: "Authorization",
			wantPrefix: "Bearer",
		},
		{
			name:       "returns oauth2 when X-OIDC-Issuer header absent",
			wwwAuth:    `Bearer realm="api"`,
			wantType:   "oauth2",
			wantHeader: "Authorization",
			wantPrefix: "Bearer",
		},
		{
			name:    "detects OIDC with extra Bearer attributes and issuer",
			wwwAuth: `Bearer realm="oidc", scope="openid"`,
			extraHeaders: map[string]string{
				"X-OIDC-Issuer": "https://accounts.example.com",
			},
			wantType:   "oidc",
			wantHeader: "Authorization",
			wantPrefix: "Bearer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("WWW-Authenticate", tt.wwwAuth)
				for k, v := range tt.extraHeaders {
					w.Header().Set(k, v)
				}
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer server.Close()

			detector := &AuthDetector{}
			resp, err := server.Client().Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close()

			auth := detector.Detect(resp)
			require.NotNil(t, auth)
			assert.Equal(t, tt.wantType, string(auth.Type))
			if tt.wantHeader != "" {
				assert.Equal(t, tt.wantHeader, auth.Header)
			}
			if tt.wantPrefix != "" {
				assert.Equal(t, tt.wantPrefix, auth.Prefix)
			}
		})
	}
}

func TestAuthDetector_UnknownAuthScheme(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value string
	}{
		{name: "returns unknown for Digest auth scheme", value: `Digest realm="protected"`},
		{name: "returns unknown for NTLM auth scheme", value: `NTLM`},
		{name: "returns unknown for Negotiate auth scheme", value: `Negotiate`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("WWW-Authenticate", tt.value)
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
		})
	}
}

func TestAuthDetector_NilResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		resp     *http.Response
		wantNil  bool
		wantType string
	}{
		{name: "nil response returns nil", resp: nil, wantNil: true},
		{
			name:     "zero-value response returns none for status 0",
			resp:     &http.Response{StatusCode: 200},
			wantNil:  false,
			wantType: "none",
		},
		{
			name:     "response with 302 redirect returns unknown",
			resp:     &http.Response{StatusCode: http.StatusFound},
			wantNil:  false,
			wantType: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			detector := &AuthDetector{}
			auth := detector.Detect(tt.resp)
			if tt.wantNil {
				assert.Nil(t, auth)
			} else {
				require.NotNil(t, auth)
				assert.Equal(t, tt.wantType, string(auth.Type))
			}
		})
	}
}
