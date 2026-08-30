package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

func TestCookieSecureEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  AuthConfig
		want bool
	}{
		{name: "nil defaults to true", cfg: AuthConfig{}, want: true},
		{name: "explicit true", cfg: AuthConfig{CookieSecure: new(true)}, want: true},
		{name: "explicit false", cfg: AuthConfig{CookieSecure: new(false)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.cookieSecureEnabled(); got != tt.want {
				t.Errorf("cookieSecureEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBruteForceEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  BruteForceConfig
		want bool
	}{
		{name: "nil defaults to true", cfg: BruteForceConfig{}, want: true},
		{name: "explicit true", cfg: BruteForceConfig{Enabled: new(true)}, want: true},
		{name: "explicit false", cfg: BruteForceConfig{Enabled: new(false)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cfg.bruteForceEnabled(); got != tt.want {
				t.Errorf("bruteForceEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBruteForceApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cfg            BruteForceConfig
		wantMax        int64
		wantLockout    int64
		wantLockoutDur string
		wantWindowDur  string
	}{
		{
			name:           "zeros filled",
			cfg:            BruteForceConfig{},
			wantMax:        5,
			wantLockout:    10,
			wantLockoutDur: "15m",
			wantWindowDur:  "15m",
		},
		{
			name: "explicit values preserved",
			cfg: BruteForceConfig{
				MaxAttempts:     3,
				LockoutAttempts: 7,
				LockoutDuration: "30m",
				WindowDuration:  "10m",
			},
			wantMax:        3,
			wantLockout:    7,
			wantLockoutDur: "30m",
			wantWindowDur:  "10m",
		},
		{
			name: "negative treated as zero then defaulted",
			cfg: BruteForceConfig{
				MaxAttempts:     -1,
				LockoutAttempts: -1,
			},
			wantMax:        5,
			wantLockout:    10,
			wantLockoutDur: "15m",
			wantWindowDur:  "15m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := tt.cfg
			cfg.applyDefaults()
			if cfg.MaxAttempts != tt.wantMax {
				t.Errorf("MaxAttempts = %d, want %d", cfg.MaxAttempts, tt.wantMax)
			}
			if cfg.LockoutAttempts != tt.wantLockout {
				t.Errorf("LockoutAttempts = %d, want %d", cfg.LockoutAttempts, tt.wantLockout)
			}
			if cfg.LockoutDuration != tt.wantLockoutDur {
				t.Errorf("LockoutDuration = %q, want %q", cfg.LockoutDuration, tt.wantLockoutDur)
			}
			if cfg.WindowDuration != tt.wantWindowDur {
				t.Errorf("WindowDuration = %q, want %q", cfg.WindowDuration, tt.wantWindowDur)
			}
		})
	}
}

func TestSetLoginRateLimiterCache_BruteForceDefaultOn(t *testing.T) {
	tests := []struct {
		name      string
		bf        BruteForceConfig
		wantLimit bool
	}{
		{name: "omitted enabled creates limiter", bf: BruteForceConfig{}, wantLimit: true},
		{name: "explicit false skips limiter", bf: BruteForceConfig{Enabled: new(false)}, wantLimit: false},
		{name: "explicit true creates limiter", bf: BruteForceConfig{Enabled: new(true)}, wantLimit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevConfig := config
			prevLimiter := loginLimiter
			prevStore := loginLimiterStore
			prevHandler := handler
			t.Cleanup(func() {
				config = prevConfig
				loginLimiter = prevLimiter
				loginLimiterStore = prevStore
				handler = prevHandler
			})

			handler = moduleHandler{initialized: true}
			config = configType{
				Enabled: true,
				Auth: AuthConfig{
					Username:   "admin",
					Password:   "flowbot-dev-pass",
					BruteForce: tt.bf,
				},
			}
			loginLimiter = nil
			loginLimiterStore = nil
			SetLoginRateLimiterCache(nil)
			// nil store must not wire even when auth would enable protection
			if loginLimiter != nil {
				t.Fatal("expected loginLimiter to remain nil without a store")
			}

			store := &cache.RedisStore{}
			SetLoginRateLimiterCache(store)
			if tt.wantLimit && loginLimiter == nil {
				t.Fatal("expected loginLimiter to be set")
			}
			if !tt.wantLimit && loginLimiter != nil {
				t.Fatal("expected loginLimiter to remain nil")
			}
		})
	}
}

func TestWireLoginRateLimiter_WaitsForInit(t *testing.T) {
	tests := []struct {
		name        string
		initialized bool
		wantLimit   bool
	}{
		{name: "before init does not wire", initialized: false, wantLimit: false},
		{name: "after init wires with defaults", initialized: true, wantLimit: true},
		{name: "init then rewire", initialized: true, wantLimit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevConfig := config
			prevLimiter := loginLimiter
			prevStore := loginLimiterStore
			prevHandler := handler
			t.Cleanup(func() {
				config = prevConfig
				loginLimiter = prevLimiter
				loginLimiterStore = prevStore
				handler = prevHandler
			})

			loginLimiter = nil
			loginLimiterStore = &cache.RedisStore{}
			handler = moduleHandler{initialized: tt.initialized}
			config = configType{
				Enabled: true,
				Auth: AuthConfig{
					Username: "admin",
					Password: "flowbot-dev-pass",
				},
			}
			wireLoginRateLimiter()
			if tt.wantLimit && loginLimiter == nil {
				t.Fatal("expected limiter")
			}
			if !tt.wantLimit && loginLimiter != nil {
				t.Fatal("expected no limiter before init")
			}
		})
	}
}

func TestAuthenticateWebRedirect(t *testing.T) {
	fullSession := map[string]any{"uid": "user-admin", "topic": "web", "kind": webauth.KindFull, "scopes": []any{"admin:*"}}
	tests := []struct {
		name             string
		cookieToken      string
		seedToken        func(t *testing.T, client *store.Client)
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:        "valid hashed token allows access to configs",
			cookieToken: "valid-token",
			seedToken: func(t *testing.T, client *store.Client) {
				seedTestAccessToken(t, client, "valid-token", fullSession, time.Now().Add(time.Hour))
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "Configs",
		},
		{
			name:             "no cookie redirects to login",
			cookieToken:      "",
			wantStatus:       http.StatusSeeOther,
			wantBodyContains: "",
		},
		{
			name:        "invalid token redirects to login",
			cookieToken: "bad-token",
			wantStatus:  http.StatusSeeOther,
		},
		{
			name:        "expired token redirects to login",
			cookieToken: "expired-token",
			seedToken: func(t *testing.T, client *store.Client) {
				seedTestAccessToken(t, client, "expired-token", fullSession, time.Now().Add(-time.Hour))
			},
			wantStatus: http.StatusSeeOther,
		},
		{
			name:        "token without scopes redirects to login",
			cookieToken: "no-scopes-token",
			seedToken: func(t *testing.T, client *store.Client) {
				seedTestAccessToken(t, client, "no-scopes-token", map[string]any{"uid": "user-admin", "topic": "web"}, time.Now().Add(time.Hour))
			},
			wantStatus: http.StatusSeeOther,
		},
		{
			name:        "legacy session without kind redirects to login",
			cookieToken: "legacy-no-kind",
			seedToken: func(t *testing.T, client *store.Client) {
				seedTestAccessToken(t, client, "legacy-no-kind", map[string]any{"uid": "user-admin", "topic": "web", "scopes": []any{"admin:*"}}, time.Now().Add(time.Hour))
			},
			wantStatus: http.StatusSeeOther,
		},
		{
			name:        "legacy plaintext token migrates and allows access",
			cookieToken: "legacy-plain-token",
			seedToken: func(t *testing.T, client *store.Client) {
				seedLegacyPlaintextAccessToken(t, client, "legacy-plain-token", fullSession, time.Now().Add(time.Hour))
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "Configs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp(t)
			if tt.seedToken != nil {
				tt.seedToken(t, ts.dbClient)
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs", http.NoBody)
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookieToken})
				AttachCSRFForTest(req)
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBodyContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBodyContains) {
					t.Errorf("want body containing %q", tt.wantBodyContains)
				}
			}
		})
	}
}

func TestSlideFullSession(t *testing.T) {
	fullWeb := testFullWebSessionParams("testuser")
	tests := []struct {
		name       string
		path       string
		token      string
		remaining  time.Duration
		params     map[string]any
		cookie     bool
		header     bool
		wantStatus int
		wantSlide  bool
	}{
		{
			name:       "cookie full web with 1h remaining extends to 24h",
			token:      "slide-stale-token",
			remaining:  time.Hour,
			params:     fullWeb,
			cookie:     true,
			wantStatus: http.StatusOK,
			wantSlide:  true,
		},
		{
			name:       "cookie full web with 24h remaining skips persist",
			token:      "slide-fresh-token",
			remaining:  webauth.FullSessionTTL,
			params:     fullWeb,
			cookie:     true,
			wantStatus: http.StatusOK,
			wantSlide:  false,
		},
		{
			name:       "header only full web does not slide",
			token:      "slide-header-token",
			remaining:  time.Hour,
			params:     fullWeb,
			header:     true,
			wantStatus: http.StatusOK,
			wantSlide:  false,
		},
		{
			name:      "cookie full session on other topic does not slide",
			token:     "slide-cli-token",
			remaining: time.Hour,
			params: map[string]any{
				"uid": "testuser", "topic": "cli", "kind": webauth.KindFull, "scopes": []any{"admin:*"},
			},
			cookie:     true,
			wantStatus: http.StatusOK,
			wantSlide:  false,
		},
		{
			name:       "pipeline page without authenticateWeb still slides",
			path:       "/service/web/pipelines",
			token:      "slide-pipeline-token",
			remaining:  time.Hour,
			params:     fullWeb,
			cookie:     true,
			wantStatus: http.StatusOK,
			wantSlide:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			originalExp := time.Now().Add(tt.remaining)
			seedTestAccessToken(t, ts.dbClient, tt.token, tt.params, originalExp)

			path := tt.path
			if path == "" {
				path = "/service/web/session-badge"
			}
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.token})
				AttachCSRFForTest(req)
			}
			if tt.header {
				req.Header.Set("X-AccessToken", tt.token)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			gotCookie := accessTokenSetCookie(resp)
			if tt.wantSlide {
				assert.Contains(t, gotCookie, "accessToken="+tt.token)
				assert.Contains(t, gotCookie, "max-age=86400")
			} else {
				assert.Empty(t, gotCookie)
			}

			p, err := route.LookupAccessToken(context.Background(), tt.token)
			require.NoError(t, err)
			if tt.wantSlide {
				assert.WithinDuration(t, time.Now().Add(webauth.FullSessionTTL), p.ExpiredAt, 3*time.Second)
			} else {
				assert.WithinDuration(t, originalExp, p.ExpiredAt, time.Second)
			}
		})
	}
}

func accessTokenSetCookie(resp *http.Response) string {
	for _, c := range resp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(c, "accessToken=") && !strings.Contains(c, "Max-Age=0") {
			return c
		}
	}
	return ""
}
