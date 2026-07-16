package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestNewEngine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  homelab.DiscoveryConfig
		wantNil bool
	}{
		{
			name:    "disabled probing returns nil",
			config:  homelab.DiscoveryConfig{ProbeEnabled: false},
			wantNil: true,
		},
		{
			name: "enabled probing returns engine",
			config: homelab.DiscoveryConfig{
				ProbeEnabled: true,
				ProbeTimeout: time.Second,
			},
			wantNil: false,
		},
		{
			name: "enabled with zero timeout still returns engine",
			config: homelab.DiscoveryConfig{
				ProbeEnabled: true,
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewEngine(tt.config)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.NotNil(t, got.probe)
		})
	}
}

func TestNewHTTPProbeAndProbeEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		handler        http.HandlerFunc
		emptyURL       bool
		wantNil        bool
		wantHealthPath string
		wantAuthType   homelab.AuthType
	}{
		{
			name:     "empty url returns nil",
			emptyURL: true,
			wantNil:  true,
		},
		{
			name: "discovers health and bearer auth",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
					w.WriteHeader(http.StatusUnauthorized)
				case "/health":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"status":"ok"}`))
				case "/.well-known/openid-configuration":
					w.WriteHeader(http.StatusNotFound)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantHealthPath: "/health",
			wantAuthType:   homelab.AuthOAuth2,
		},
		{
			name: "oidc discovery upgrades unknown auth",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`<html><title>App</title></html>`))
				case "/.well-known/openid-configuration":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"issuer":"http://example"}`))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantAuthType: homelab.AuthOIDC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewHTTPProbe(2 * time.Second)
			require.NotNil(t, p)

			if tt.emptyURL {
				assert.Nil(t, p.ProbeEndpoint(context.Background(), ""))
				return
			}

			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			result := p.ProbeEndpoint(context.Background(), srv.URL)
			require.NotNil(t, result)
			assert.Equal(t, strings.TrimRight(srv.URL, "/"), result.BaseURL)
			if tt.wantHealthPath != "" {
				assert.Contains(t, result.HealthURL, tt.wantHealthPath)
			}
			if tt.wantAuthType != "" {
				require.NotNil(t, result.Auth)
				assert.Equal(t, tt.wantAuthType, result.Auth.Type)
			}
		})
	}
}
