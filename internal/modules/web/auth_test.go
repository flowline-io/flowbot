package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/types"
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
	tests := []struct {
		name             string
		cookieToken      string
		paramGetFn       func(ctx context.Context, flag string) (gen.Parameter, error)
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:        "valid hashed token allows access to configs",
			cookieToken: "valid-token",
			paramGetFn: func(_ context.Context, flag string) (gen.Parameter, error) {
				if flag != auth.HashToken("valid-token") {
					return gen.Parameter{}, types.ErrNotFound
				}
				return gen.Parameter{
					ID:        1,
					Flag:      flag,
					Params:    map[string]any{"uid": "user-admin", "topic": "web", "scopes": []any{"admin:*"}},
					ExpiredAt: time.Now().Add(time.Hour),
				}, nil
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
			paramGetFn: func(_ context.Context, _ string) (gen.Parameter, error) {
				return gen.Parameter{}, types.ErrNotFound
			},
			wantStatus:       http.StatusSeeOther,
			wantBodyContains: "",
		},
		{
			name:        "expired token redirects to login",
			cookieToken: "expired-token",
			paramGetFn: func(_ context.Context, flag string) (gen.Parameter, error) {
				if flag != auth.HashToken("expired-token") {
					return gen.Parameter{}, types.ErrNotFound
				}
				return gen.Parameter{
					ID:        2,
					Flag:      flag,
					Params:    map[string]any{"uid": "user-admin", "topic": "web", "scopes": []any{"admin:*"}},
					ExpiredAt: time.Now().Add(-time.Hour),
				}, nil
			},
			wantStatus:       http.StatusSeeOther,
			wantBodyContains: "",
		},
		{
			name:        "token without scopes redirects to login",
			cookieToken: "no-scopes-token",
			paramGetFn: func(_ context.Context, flag string) (gen.Parameter, error) {
				if flag != auth.HashToken("no-scopes-token") {
					return gen.Parameter{}, types.ErrNotFound
				}
				return gen.Parameter{
					ID:        4,
					Flag:      flag,
					Params:    map[string]any{"uid": "user-admin", "topic": "web"},
					ExpiredAt: time.Now().Add(time.Hour),
				}, nil
			},
			wantStatus:       http.StatusSeeOther,
			wantBodyContains: "",
		},
		{
			name:        "legacy plaintext token migrates and allows access",
			cookieToken: "legacy-plain-token",
			paramGetFn: func(_ context.Context, flag string) (gen.Parameter, error) {
				if flag == "legacy-plain-token" {
					return gen.Parameter{
						ID:        3,
						Flag:      flag,
						Params:    map[string]any{"uid": "user-admin", "topic": "web", "scopes": []any{"admin:*"}},
						ExpiredAt: time.Now().Add(time.Hour),
					}, nil
				}
				return gen.Parameter{}, types.ErrNotFound
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "Configs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			if tt.paramGetFn != nil {
				ts.paramGetFn = tt.paramGetFn
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
