package media

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestGetIdFromUrl(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		url      string
		serveURL string
		want     types.Uid
	}{
		{name: "extracts id from serve path", url: "/files/abc123", serveURL: "/files/", want: types.Uid("abc123")},
		{name: "wrong directory returns zero uid", url: "/other/abc123", serveURL: "/files/", want: types.ZeroUid},
		{name: "filename without pattern returns empty uid", url: "/files/!!!", serveURL: "/files/", want: types.Uid("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, GetIdFromUrl(tt.url, tt.serveURL))
		})
	}
}

func TestCORSHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		method          string
		origin          string
		requestMethod   string
		allowedOrigins  []string
		serve           bool
		wantStatus      int
		wantAllowOrigin string
	}{
		{
			name:       "non-options request passes through",
			method:     http.MethodGet,
			wantStatus: 0,
		},
		{
			name:            "options with allowed origin for upload",
			method:          http.MethodOptions,
			origin:          "https://app.example.com",
			requestMethod:   http.MethodPost,
			allowedOrigins:  []string{"https://app.example.com"},
			serve:           false,
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: "https://app.example.com",
		},
		{
			name:            "options with wildcard origin",
			method:          http.MethodOptions,
			origin:          "https://any.example.com",
			requestMethod:   http.MethodGet,
			allowedOrigins:  []string{"*"},
			serve:           true,
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: "*",
		},
		{
			name:            "options rejects disallowed method",
			method:          http.MethodOptions,
			origin:          "https://app.example.com",
			requestMethod:   http.MethodDelete,
			allowedOrigins:  []string{"https://app.example.com"},
			serve:           false,
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tt.method, "/upload", http.NoBody)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.requestMethod != "" {
				req.Header.Set("Access-Control-Request-Method", tt.requestMethod)
			}
			headers, status := CORSHandler(req, tt.allowedOrigins, tt.serve)
			assert.Equal(t, tt.wantStatus, status)
			if tt.wantAllowOrigin != "" {
				require.NotNil(t, headers)
				assert.Equal(t, []string{tt.wantAllowOrigin}, headers["Access-Control-Allow-Origin"])
			}
		})
	}
}
