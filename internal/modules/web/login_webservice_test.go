package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
)

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

// assertCookie verifies that the response sets or does not set an accessToken cookie.
func assertCookie(t *testing.T, resp *http.Response, wantSet bool) {
	t.Helper()
	for _, c := range resp.Header.Values("Set-Cookie") {
		if strings.Contains(c, "accessToken=") && !strings.Contains(c, "Max-Age=0") {
			if !wantSet {
				t.Error("accessToken cookie should NOT be set")
			}
			return
		}
	}
	if wantSet {
		t.Error("expected accessToken cookie to be set")
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
			name:          "wrong password shows error",
			username:      "admin",
			password:      "wrong",
			wantStatus:    http.StatusOK,
			wantContains:  "Invalid username or password",
			wantCookieSet: false,
		},
		{
			name:          "empty username shows error",
			username:      "",
			password:      "admin",
			wantStatus:    http.StatusOK,
			wantContains:  "Invalid username or password",
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
			name:          "param set error renders error",
			username:      "admin",
			password:      "admin",
			paramSetErr:   fmt.Errorf("db down"),
			wantStatus:    http.StatusOK,
			wantContains:  "Internal error",
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
			assertCookie(t, resp, tt.wantCookieSet)
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q, got %q", tt.wantContains, string(body))
				}
			}
		})
	}
}

func TestLoginSubmitReturnsFragmentOnError(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		password     string
		notContains  []string
		wantContains string
	}{
		{
			name:         "wrong password returns form fragment not full page",
			username:     "admin",
			password:     "wrong",
			notContains:  []string{"<!DOCTYPE", "<html", "<body", "<nav"},
			wantContains: "Invalid username or password",
		},
		{
			name:         "empty credentials returns form fragment not full page",
			username:     "",
			password:     "",
			notContains:  []string{"<!DOCTYPE", "<html", "<body", "<nav"},
			wantContains: "Invalid username or password",
		},
		{
			name:         "param set error returns form fragment not full page",
			username:     "admin",
			password:     "admin",
			notContains:  []string{"<!DOCTYPE", "<html", "<body", "<nav"},
			wantContains: "Internal error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			if tt.username == "admin" && tt.password == "admin" {
				ts.paramSetFn = func(_ context.Context, _ string, _ types.KV, _ time.Time) error {
					return fmt.Errorf("db down")
				}
			}
			form := url.Values{}
			form.Set("username", tt.username)
			form.Set("password", tt.password)
			req := httptest.NewRequest(http.MethodPost, "/service/web/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			for _, s := range tt.notContains {
				if strings.Contains(bodyStr, s) {
					t.Errorf("response should NOT contain %q but it did", s)
				}
			}
			if !strings.Contains(bodyStr, tt.wantContains) {
				t.Errorf("wanted body containing %q", tt.wantContains)
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
			name:        "logout with cookie sets HX-Redirect",
			cookieToken: "token-to-delete",
			wantStatus:  http.StatusOK,
			wantDel:     true,
		},
		{
			name:        "logout without cookie still sets HX-Redirect",
			cookieToken: "",
			wantStatus:  http.StatusOK,
			wantDel:     false,
		},
		{
			name:        "logout ignores param delete error and sets HX-Redirect",
			cookieToken: "error-token",
			wantStatus:  http.StatusOK,
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
			hxRedirect := resp.Header.Get("HX-Redirect")
			if hxRedirect != "/service/web/login" {
				t.Errorf("want HX-Redirect /service/web/login, got %q", hxRedirect)
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

func TestLoginSubmitRateLimitLocked(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*mockRateLimitStore)
		wantStatus   int
		wantContains string
	}{
		{
			name: "locked IP returns lockout message",
			setup: func(m *mockRateLimitStore) {
				m.setLock("0.0.0.0")
			},
			wantStatus:   http.StatusOK,
			wantContains: "Account temporarily locked",
		},
		{
			name: "below threshold shows normal error on wrong password",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("0.0.0.0", 2)
			},
			wantStatus:   http.StatusOK,
			wantContains: "Invalid username or password",
		},
		{
			name: "some failed attempts not locked shows error",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("0.0.0.0", 4)
			},
			wantStatus:   http.StatusOK,
			wantContains: "Invalid username or password",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, mockStore := setupTestAppWithRateLimiter()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{}; loginLimiter = nil }()
			if tt.setup != nil {
				tt.setup(mockStore)
			}
			form := url.Values{}
			form.Set("username", "admin")
			form.Set("password", "wrong")
			req := httptest.NewRequest(http.MethodPost, "/service/web/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q, got %q", tt.wantContains, string(body))
			}
		})
	}
}

func TestLoginSubmitSuccessClearsRateLimit(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*mockRateLimitStore)
		wantStatus     int
		wantHXRedirect string
		wantCookieSet  bool
	}{
		{
			name: "success clears existing failure count",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("0.0.0.0", 4)
			},
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/configs",
			wantCookieSet:  true,
		},
		{
			name: "success with no prior failures works normally",
			setup: func(m *mockRateLimitStore) {
				m.setAttempts("0.0.0.0", 1)
			},
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/configs",
			wantCookieSet:  true,
		},
		{
			name:           "success with clean state works normally",
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/configs",
			wantCookieSet:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, mockStore := setupTestAppWithRateLimiter()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{}; loginLimiter = nil }()
			if tt.setup != nil {
				tt.setup(mockStore)
			}
			form := url.Values{}
			form.Set("username", "admin")
			form.Set("password", "admin")
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
			assertCookie(t, resp, tt.wantCookieSet)
			attemptVal, _ := mockStore.GetInt64(context.Background(), attemptKey("0.0.0.0"))
			if attemptVal != 0 {
				t.Errorf("expected attempts cleared, got %d", attemptVal)
			}
			exists, _ := mockStore.Exists(context.Background(), lockKey("0.0.0.0"))
			if exists {
				t.Error("expected lock cleared")
			}
		})
	}
}
