package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestRegister(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "register should not panic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				Register()
			})
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		jsonCfg string
		wantErr bool
	}{
		{
			name:    "enabled true succeeds",
			jsonCfg: `{"enabled": true}`,
			wantErr: false,
		},
		{
			name:    "disabled skips initialization",
			jsonCfg: `{"enabled": false}`,
			wantErr: false,
		},
		{
			name:    "invalid json returns error",
			jsonCfg: `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &moduleHandler{}
			err := h.Init(json.RawMessage(tt.jsonCfg))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Reset handler state for subsequent tests
			handler = moduleHandler{}
			config = configType{}
		})
	}
}

func TestIsReady(t *testing.T) {
	tests := []struct {
		name        string
		initialized bool
		want        bool
	}{
		{
			name:        "ready after init",
			initialized: true,
			want:        true,
		},
		{
			name:        "not ready before init",
			initialized: false,
			want:        false,
		},
		{
			name:        "not ready when disabled",
			initialized: false,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: tt.initialized}
			assert.Equal(t, tt.want, handler.IsReady())
			handler = moduleHandler{}
		})
	}
}

func TestConfigsPage(t *testing.T) {
	tests := []struct {
		name, wantContains string
		storeConfigs       []model.ConfigItem
		storeErr           error
		wantStatus         int
	}{
		{name: "renders page with configs", storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1")}, wantStatus: http.StatusOK, wantContains: "k1"},
		{name: "renders page with empty list", storeConfigs: []model.ConfigItem{}, wantStatus: http.StatusOK, wantContains: "Configs"},
		{name: "store error returns 500", storeErr: fmt.Errorf("db down"), wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.configs = tt.storeConfigs
			if tt.storeErr != nil {
				ts.configErr = tt.storeErr
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q", tt.wantContains)
				}
			}
		})
	}
}

func TestListConfigs(t *testing.T) {
	tests := []struct {
		name, wantContains string
		storeConfigs       []model.ConfigItem
		wantStatus         int
	}{
		{name: "renders config table", storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1")}, wantStatus: http.StatusOK, wantContains: "k1"},
		{name: "renders empty state", storeConfigs: []model.ConfigItem{}, wantStatus: http.StatusOK, wantContains: "No configs"},
		{name: "renders multiple rows", storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1"), createTestConfig("u2", "t2", "k2")}, wantStatus: http.StatusOK, wantContains: "k2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.configs = tt.storeConfigs
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs/list", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q", tt.wantContains)
				}
			}
		})
	}
}

func TestDeleteConfig(t *testing.T) {
	tests := []struct {
		name       string
		delErr     error
		wantStatus int
	}{
		{name: "delete returns 200 on success", wantStatus: http.StatusOK},
		{name: "delete returns 500 on store error", delErr: fmt.Errorf("db down"), wantStatus: http.StatusInternalServerError},
		{name: "delete non-existent returns 404", delErr: types.ErrNotFound, wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			if tt.delErr != nil {
				ts.delConfigFn = func(_ types.Uid, _ string, _ string) error { return tt.delErr }
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodDelete, "/service/web/configs/u1/t1/k1", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name       string
		getFn      func(uid types.Uid, topic, key string) (types.KV, error)
		wantStatus int
	}{
		{name: "existing config returns row", getFn: func(_ types.Uid, _ string, _ string) (types.KV, error) { return types.KV{"v": "foo"}, nil }, wantStatus: http.StatusOK},
		{name: "not found returns 404", getFn: func(_ types.Uid, _ string, _ string) (types.KV, error) { return nil, types.ErrNotFound }, wantStatus: http.StatusNotFound},
		{name: "store error returns 500", getFn: func(_ types.Uid, _ string, _ string) (types.KV, error) { return nil, fmt.Errorf("db down") }, wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.getConfigFn = tt.getFn
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs/u1/t1/k1", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestLoginPage(t *testing.T) {
	tests := []struct {
		name         string
		cookieToken  string
		paramGetFn   func(ctx context.Context, flag string) (gen.Parameter, error)
		wantStatus   int
		wantContains string
		wantLocation string
	}{
		{
			name:         "no cookie renders login form",
			wantStatus:   http.StatusOK,
			wantContains: "Login",
		},
		{
			name:         "with valid cookie redirects to configs",
			cookieToken:  "valid-token",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/service/web/configs",
		},
		{
			name:        "with expired token renders login form",
			cookieToken: "expired-token",
			paramGetFn: func(_ context.Context, flag string) (gen.Parameter, error) {
				return gen.Parameter{
					ID:        1,
					Flag:      flag,
					Params:    map[string]any{"uid": "testuser", "topic": "test"},
					ExpiredAt: time.Now().Add(-time.Hour),
				}, nil
			},
			wantStatus:   http.StatusOK,
			wantContains: "Login",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			if tt.paramGetFn != nil {
				ts.paramGetFn = tt.paramGetFn
			}
			req := httptest.NewRequest(http.MethodGet, "/service/web/login", nil)
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookieToken})
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantLocation != "" {
				loc := resp.Header.Get("Location")
				if loc != tt.wantLocation {
					t.Errorf("want location %q, got %q", tt.wantLocation, loc)
				}
			}
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q", tt.wantContains)
				}
			}
		})
	}
}

func TestLoginSubmit(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		password       string
		nextVal        string
		paramSetErr    error
		wantStatus     int
		wantContains   string
		wantHXRedirect string
		wantCookieSet  bool
	}{
		{
			name:           "correct credentials returns redirect",
			username:       "admin",
			password:       "admin",
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/configs",
			wantCookieSet:  true,
		},
		{
			name:         "wrong password shows error",
			username:     "admin",
			password:     "wrong",
			wantStatus:   http.StatusOK,
			wantContains: "Invalid username or password",
			wantCookieSet: false,
		},
		{
			name:         "empty username shows error",
			username:     "",
			password:     "admin",
			wantStatus:   http.StatusOK,
			wantContains: "Invalid username or password",
			wantCookieSet: false,
		},
		{
			name:           "correct credentials with valid next redirects",
			username:       "admin",
			password:       "admin",
			nextVal:        "/service/web/configs?page=2",
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/configs?page=2",
			wantCookieSet:  true,
		},
		{
			name:           "correct credentials with external next falls back",
			username:       "admin",
			password:       "admin",
			nextVal:        "https://evil.com",
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/configs",
			wantCookieSet:  true,
		},
		{
			name:        "param set error renders error",
			username:    "admin",
			password:    "admin",
			paramSetErr: fmt.Errorf("db down"),
			wantStatus:  http.StatusOK,
			wantContains: "Internal error",
			wantCookieSet: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			if tt.paramSetErr != nil {
				ts.paramSetFn = func(_ context.Context, _ string, _ types.KV, _ time.Time) error {
					return tt.paramSetErr
				}
			}
			form := url.Values{}
			form.Set("username", tt.username)
			form.Set("password", tt.password)
			if tt.nextVal != "" {
				form.Set("next", tt.nextVal)
			}
			req := httptest.NewRequest(http.MethodPost, "/service/web/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantHXRedirect != "" {
				got := resp.Header.Get("HX-Redirect")
				if got != tt.wantHXRedirect {
					t.Errorf("want HX-Redirect %q, got %q", tt.wantHXRedirect, got)
				}
			}
			if tt.wantCookieSet {
				found := false
				for _, c := range resp.Header.Values("Set-Cookie") {
					if strings.Contains(c, "accessToken=") && !strings.Contains(c, "Max-Age=0") {
						found = true
					}
				}
				if !found {
					t.Error("expected accessToken cookie to be set")
				}
			} else {
				for _, c := range resp.Header.Values("Set-Cookie") {
					if strings.Contains(c, "accessToken=") && !strings.Contains(c, "Max-Age=0") {
						t.Error("accessToken cookie should NOT be set")
					}
				}
			}
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q, got %q", tt.wantContains, string(body))
				}
			}
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name        string
		cookieToken string
		wantStatus  int
		wantDel     bool
	}{
		{
			name:        "logout with cookie deletes it",
			cookieToken: "token-to-delete",
			wantStatus:  http.StatusSeeOther,
			wantDel:     true,
		},
		{
			name:        "logout without cookie still redirects",
			cookieToken: "",
			wantStatus:  http.StatusSeeOther,
			wantDel:     false,
		},
		{
			name:        "logout ignores param delete error",
			cookieToken: "error-token",
			wantStatus:  http.StatusSeeOther,
			wantDel:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			deletedFlag := ""
			ts.paramDelFn = func(_ context.Context, flag string) error {
				deletedFlag = flag
				if flag == "error-token" {
					return fmt.Errorf("db error")
				}
				return nil
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodPost, "/service/web/logout", nil)
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookieToken})
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			loc := resp.Header.Get("Location")
			if loc != "/service/web/login" {
				t.Errorf("want location /service/web/login, got %q", loc)
			}
			if tt.wantDel && deletedFlag == "" {
				t.Error("expected ParameterDelete to be called")
			}
			if !tt.wantDel && deletedFlag != "" {
				t.Error("expected ParameterDelete NOT to be called for empty cookie")
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
			name:        "valid token allows access to configs",
			cookieToken: "valid-token",
			paramGetFn: func(_ context.Context, flag string) (gen.Parameter, error) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			if tt.paramGetFn != nil {
				ts.paramGetFn = tt.paramGetFn
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/configs", nil)
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
