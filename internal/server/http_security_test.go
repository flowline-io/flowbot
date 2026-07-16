package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/config"
)

func TestMatchHTTPAllowOrigin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		allowed []string
		origin  string
		want    bool
	}{
		{name: "empty whitelist rejects any origin", allowed: nil, origin: "https://evil.example", want: false},
		{name: "empty whitelist rejects empty origin", allowed: []string{}, origin: "", want: false},
		{name: "exact match allows origin", allowed: []string{"https://app.example"}, origin: "https://app.example", want: true},
		{name: "case insensitive match", allowed: []string{"https://App.Example"}, origin: "https://app.example", want: true},
		{name: "non-matching origin rejected", allowed: []string{"https://app.example"}, origin: "https://other.example", want: false},
		{name: "star allows any non-empty origin", allowed: []string{"*"}, origin: "https://any.example", want: true},
		{name: "star rejects empty origin", allowed: []string{"*"}, origin: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, matchHTTPAllowOrigin(tt.allowed, tt.origin))
		})
	}
}

func TestHTTPRateLimitDefaults(t *testing.T) {
	tests := []struct {
		name       string
		max        int
		expiration time.Duration
		wantMax    int
		wantExp    time.Duration
	}{
		{name: "zero uses defaults", max: 0, expiration: 0, wantMax: 200, wantExp: 10 * time.Second},
		{name: "negative max uses default", max: -1, expiration: time.Second, wantMax: 200, wantExp: time.Second},
		{name: "configured values used", max: 50, expiration: 5 * time.Second, wantMax: 50, wantExp: 5 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := config.App.HTTP.RateLimit
			t.Cleanup(func() { config.App.HTTP.RateLimit = prev })
			config.App.HTTP.RateLimit = config.HTTPRateLimitConfig{Max: tt.max, Expiration: tt.expiration}
			assert.Equal(t, tt.wantMax, httpRateLimitMax())
			assert.Equal(t, tt.wantExp, httpRateLimitExpiration())
		})
	}
}

func TestSecurityHeadersMiddleware_HSTS(t *testing.T) {
	tests := []struct {
		name           string
		tlsBehindProxy bool
		modules        any
		wantHSTS       bool
	}{
		{name: "HSTS omitted when tls and cookie_secure off", tlsBehindProxy: false, wantHSTS: false},
		{name: "HSTS sent when tls_behind_proxy true", tlsBehindProxy: true, wantHSTS: true},
		{
			name:           "HSTS sent when web cookie_secure omitted",
			tlsBehindProxy: false,
			modules: []map[string]any{
				{"name": "web", "auth": map[string]any{}},
			},
			wantHSTS: true,
		},
		{
			name:           "HSTS omitted when web cookie_secure false",
			tlsBehindProxy: false,
			modules: []map[string]any{
				{"name": "web", "auth": map[string]any{"cookie_secure": false}},
			},
			wantHSTS: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevHTTP := config.App.HTTP
			prevModules := config.App.Modules
			t.Cleanup(func() {
				config.App.HTTP = prevHTTP
				config.App.Modules = prevModules
			})
			config.App.HTTP.TLSBehindProxy = tt.tlsBehindProxy
			config.App.Modules = tt.modules

			app := fiber.New()
			app.Use(securityHeadersMiddleware)
			app.Get("/probe", func(c fiber.Ctx) error { return c.SendString("ok") })

			req := httptest.NewRequest(http.MethodGet, "/probe", http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			hsts := resp.Header.Get(fiber.HeaderStrictTransportSecurity)
			if tt.wantHSTS {
				assert.Contains(t, hsts, "max-age=")
			} else {
				assert.Empty(t, hsts)
			}
			assert.Equal(t, "nosniff", resp.Header.Get(fiber.HeaderXContentTypeOptions))
			assert.Equal(t, "DENY", resp.Header.Get(fiber.HeaderXFrameOptions))
		})
	}
}

func TestCorsAllowCredentials(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		allowed []string
		want    bool
	}{
		{name: "empty disables credentials", allowed: nil, want: false},
		{name: "star disables credentials", allowed: []string{"*"}, want: false},
		{name: "explicit origin enables credentials", allowed: []string{"https://app.example"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, corsAllowCredentials(tt.allowed))
		})
	}
}

func TestCORSAllowOriginsWhitelist(t *testing.T) {
	tests := []struct {
		name         string
		allowOrigins []string
		origin       string
		wantACAO     string
		wantCreds    bool
	}{
		{
			name:         "empty whitelist does not reflect origin",
			allowOrigins: nil,
			origin:       "https://evil.example",
			wantACAO:     "",
			wantCreds:    false,
		},
		{
			name:         "matching origin reflected with credentials",
			allowOrigins: []string{"https://app.example"},
			origin:       "https://app.example",
			wantACAO:     "https://app.example",
			wantCreds:    true,
		},
		{
			name:         "non-matching origin not reflected",
			allowOrigins: []string{"https://app.example"},
			origin:       "https://other.example",
			wantACAO:     "",
			wantCreds:    false,
		},
		{
			name:         "star reflects origin without credentials",
			allowOrigins: []string{"*"},
			origin:       "https://any.example",
			wantACAO:     "https://any.example",
			wantCreds:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := config.App.HTTP
			t.Cleanup(func() { config.App.HTTP = prev })
			config.App.HTTP = config.HTTPConfig{
				CORS: config.HTTPCORSConfig{AllowOrigins: tt.allowOrigins},
			}

			app := newHTTPServer()
			defer app.Shutdown()
			app.Get("/cors-probe", func(c fiber.Ctx) error { return c.SendString("ok") })

			req := httptest.NewRequest(http.MethodGet, "/cors-probe", http.NoBody)
			req.Header.Set("Origin", tt.origin)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantACAO, resp.Header.Get(fiber.HeaderAccessControlAllowOrigin))
			creds := resp.Header.Get(fiber.HeaderAccessControlAllowCredentials)
			if tt.wantCreds {
				assert.Equal(t, "true", creds)
			} else {
				assert.NotEqual(t, "true", creds)
			}
		})
	}
}

func TestSecureTokenEqual(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{name: "equal tokens", a: "secret", b: "secret", want: true},
		{name: "different tokens", a: "secret", b: "other", want: false},
		{name: "different lengths", a: "short", b: "longer", want: false},
		{name: "empty equal", a: "", b: "", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, secureTokenEqual(tt.a, tt.b))
		})
	}
}

func TestMetricsAuth_BearerToken(t *testing.T) {
	tests := []struct {
		name       string
		bearerCfg  string
		authHeader string
		wantStatus int
	}{
		{name: "matching bearer allows", bearerCfg: "scrape-secret", authHeader: "Bearer scrape-secret", wantStatus: http.StatusOK},
		{name: "wrong bearer rejected", bearerCfg: "scrape-secret", authHeader: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "missing auth rejected when bearer configured", bearerCfg: "scrape-secret", authHeader: "", wantStatus: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := config.App.Metrics.BearerToken
			origDB := store.Database
			t.Cleanup(func() {
				config.App.Metrics.BearerToken = prev
				store.Database = origDB
			})
			config.App.Metrics.BearerToken = tt.bearerCfg
			store.Database = &testStoreAdapter{}

			app := fiber.New(fiber.Config{
				ErrorHandler: func(ctx fiber.Ctx, err error) error {
					return ctx.Status(fiber.StatusUnauthorized).SendString(err.Error())
				},
			})
			app.Get("/metrics", metricsAuth, func(c fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
			if tt.authHeader != "" {
				req.Header.Set(fiber.HeaderAuthorization, tt.authHeader)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
