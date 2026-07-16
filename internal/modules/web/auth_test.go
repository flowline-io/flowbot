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
