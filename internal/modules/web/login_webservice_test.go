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

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

func TestLoginPage(t *testing.T) {
	tests := []struct {
		name         string
		cookieToken  string
		paramGetFn   func(ctx context.Context, flag string) (gen.Parameter, error)
		seedAccount  bool
		wantStatus   int
		wantContains string
		wantLocation string
	}{
		{
			name:         "no cookie with account renders login form",
			seedAccount:  true,
			wantStatus:   http.StatusOK,
			wantContains: "Login",
		},
		{
			name:         "login form embeds csrf_token field",
			seedAccount:  true,
			wantStatus:   http.StatusOK,
			wantContains: `name="csrf_token"`,
		},
		{
			name:         "with valid cookie redirects to home",
			seedAccount:  true,
			cookieToken:  "valid-token",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/service/web/home",
		},
		{
			name:        "with expired token renders login form",
			seedAccount: true,
			cookieToken: "expired-token",
			paramGetFn: func(_ context.Context, flag string) (gen.Parameter, error) {
				return gen.Parameter{
					ID:        1,
					Flag:      flag,
					Params:    map[string]any{"uid": "testuser", "topic": "test", "scopes": []string{"admin:*"}},
					ExpiredAt: time.Now().Add(-time.Hour),
				}, nil
			},
			wantStatus:   http.StatusOK,
			wantContains: "Login",
		},
		{
			name:         "no accounts redirects to setup",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/service/web/setup",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts, client := setupTestAppWithDB(t)
			if tt.seedAccount {
				seedWebAccount(t, client, "admin", "flowbot-dev-pass", true)
			}
			if tt.paramGetFn != nil {
				ts.paramGetFn = tt.paramGetFn
			}
			req := httptest.NewRequest(http.MethodGet, "/service/web/login", http.NoBody)
			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: tt.cookieToken})
				AttachCSRFForTest(req)
			}
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
			require.NoError(t, err)
			require.NotNil(t, resp)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want %d got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantLocation != "" {
				loc := resp.Header.Get("Location")
				if loc != tt.wantLocation {
					t.Errorf("want Location %q got %q", tt.wantLocation, loc)
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

func assertPendingCookie(t *testing.T, resp *http.Response, wantSet bool) {
	t.Helper()
	for _, c := range resp.Header.Values("Set-Cookie") {
		if strings.Contains(c, "pendingAuth=") && !strings.Contains(c, "Max-Age=0") {
			if !wantSet {
				t.Error("pendingAuth cookie should NOT be set")
			}
			return
		}
	}
	if wantSet {
		t.Error("expected pendingAuth cookie to be set")
	}
}

func TestLoginSubmit(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		password       string
		nextVal        string
		totpEnabled    bool
		paramSetErr    error
		wantStatus     int
		wantContains   string
		wantHXRedirect string
		wantPending    bool
	}{
		{
			name:           "correct credentials with totp redirects to 2fa",
			username:       "admin",
			password:       "flowbot-dev-pass",
			totpEnabled:    true,
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/login/2fa",
			wantPending:    true,
		},
		{
			name:           "correct credentials without totp redirects to enroll",
			username:       "admin",
			password:       "flowbot-dev-pass",
			totpEnabled:    false,
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/setup/2fa",
			wantPending:    true,
		},
		{
			name:         "wrong password shows error",
			username:     "admin",
			password:     "wrong-password",
			totpEnabled:  true,
			wantStatus:   http.StatusOK,
			wantContains: "Invalid username or password",
			wantPending:  false,
		},
		{
			name:           "correct credentials with valid next preserves next on 2fa",
			username:       "admin",
			password:       "flowbot-dev-pass",
			nextVal:        "/service/web/configs?page=2",
			totpEnabled:    true,
			wantStatus:     http.StatusOK,
			wantHXRedirect: "/service/web/login/2fa?next=" + url.QueryEscape("/service/web/configs?page=2"),
			wantPending:    true,
		},
		{
			name:         "param set error renders error",
			username:     "admin",
			password:     "flowbot-dev-pass",
			totpEnabled:  true,
			paramSetErr:  fmt.Errorf("db down"),
			wantStatus:   http.StatusOK,
			wantContains: "Internal error",
			wantPending:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts, client := setupTestAppWithDB(t)
			seedWebAccount(t, client, "admin", "flowbot-dev-pass", tt.totpEnabled)
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
			AttachCSRFForTest(req)
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
			require.NoError(t, err)
			require.NotNil(t, resp)
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
			assertPendingCookie(t, resp, tt.wantPending)
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantContains) {
					t.Errorf("want body containing %q, got %q", tt.wantContains, string(body))
				}
			}
		})
	}
}

func TestLogin2FASubmit(t *testing.T) {
	app, ts, client := setupTestAppWithDB(t)
	seedWebAccount(t, client, "admin", "flowbot-dev-pass", true)

	ws := store.NewWebAccountStore(client)
	account, err := ws.GetByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := accountTOTPSecret(account.TotpSecretCiphertext, account.TotpSecretNonce)
	if err != nil {
		t.Fatal(err)
	}
	code, err := webauth.CodeAt(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	pendingToken := "pending-test-token"
	ts.paramGetFn = func(_ context.Context, flag string) (gen.Parameter, error) {
		if flag == auth.HashToken(pendingToken) || flag == pendingToken {
			return gen.Parameter{
				ID: 1, Flag: flag,
				Params: map[string]any{
					"uid": account.UID, "username": "admin", "topic": "web", "kind": webauth.KindPending2FA,
				},
				ExpiredAt: time.Now().Add(time.Minute),
			}, nil
		}
		return gen.Parameter{}, types.ErrNotFound
	}
	var storedParams types.KV
	ts.paramSetFn = func(_ context.Context, _ string, params types.KV, _ time.Time) error {
		storedParams = params
		return nil
	}

	form := url.Values{}
	form.Set("code", code)
	req := httptest.NewRequest(http.MethodPost, "/service/web/login/2fa", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: webauth.CookiePending, Value: pendingToken})
	AttachCSRFForTest(req)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	if resp.Header.Get("HX-Redirect") != "/service/web/home" {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("want redirect home, got %q body %s", resp.Header.Get("HX-Redirect"), body)
	}
	if kind, _ := storedParams.String("kind"); kind != webauth.KindFull {
		t.Fatalf("want full session kind, got %#v", storedParams)
	}
}

func TestLogin2FABackupCodeBypassesTOTPLock(t *testing.T) {
	app, ts, client := setupTestAppWithDB(t)

	mockStore := newMockRateLimitStore()
	totpLimiter = newLoginRateLimiter(mockStore, 3, 10, cache.TTL(15*time.Minute), cache.TTL(15*time.Minute))

	enc := getEncryptor()
	codes, hashes, err := enc.GenerateBackupCodes(1)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := webauth.HashPassword("flowbot-dev-pass")
	if err != nil {
		t.Fatal(err)
	}
	ws := store.NewWebAccountStore(client)
	_, err = ws.CreateFirstAccount(context.Background(), store.CreateAccountInput{
		Username:     "admin",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatal(err)
	}
	secret, err := webauth.GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	ct, nonce, err := enc.Encrypt([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.EnableTOTP(context.Background(), "admin", ct, nonce, hashes, 0); err != nil {
		t.Fatal(err)
	}
	account, err := ws.GetByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	pendingToken := "pending-backup-token"
	ts.paramGetFn = func(_ context.Context, flag string) (gen.Parameter, error) {
		if flag == auth.HashToken(pendingToken) || flag == pendingToken {
			return gen.Parameter{
				ID: 1, Flag: flag,
				Params: map[string]any{
					"uid": account.UID, "username": "admin", "topic": "web", "kind": webauth.KindPending2FA,
				},
				ExpiredAt: time.Now().Add(time.Minute),
			}, nil
		}
		return gen.Parameter{}, types.ErrNotFound
	}

	ip := "203.0.113.10"
	for range 12 {
		totpLimiter.RecordFailure(context.Background(), ip+":totp")
	}

	form := url.Values{}
	form.Set("code", codes[0])
	req := httptest.NewRequest(http.MethodPost, "/service/web/login/2fa", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", ip)
	req.AddCookie(&http.Cookie{Name: webauth.CookiePending, Value: pendingToken})
	AttachCSRFForTest(req)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	if resp.Header.Get("HX-Redirect") != "/service/web/home" {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("want redirect home with backup code under totp lock, got %q body %s", resp.Header.Get("HX-Redirect"), body)
	}
}

func TestLoginSubmitCookieAttributes(t *testing.T) {
	secureFalse := false
	app, _, client := setupTestAppWithDB(t)
	handler.authConfig = AuthConfig{CookieSecure: &secureFalse}
	config.Auth = handler.authConfig
	seedWebAccount(t, client, "admin", "flowbot-dev-pass", true)

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "flowbot-dev-pass")
	req := httptest.NewRequest(http.MethodPost, "/service/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	AttachCSRFForTest(req)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp == nil {
		t.Fatal("app.Test returned nil response")
	}
	defer resp.Body.Close()
	found := false
	for _, c := range resp.Header.Values("Set-Cookie") {
		if strings.Contains(c, "pendingAuth=") {
			found = true
			if strings.Contains(strings.ToLower(c), "secure") {
				t.Errorf("Secure should be off when cookie_secure=false: %s", c)
			}
		}
	}
	if !found {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected pendingAuth cookie, status=%d body=%s", resp.StatusCode, body)
	}
}

func TestLoginSubmitStoresHashedToken(t *testing.T) {
	app, ts, client := setupTestAppWithDB(t)
	seedWebAccount(t, client, "admin", "flowbot-dev-pass", true)
	var storedFlag string
	ts.paramSetFn = func(_ context.Context, flag string, _ types.KV, _ time.Time) error {
		storedFlag = flag
		return nil
	}
	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "flowbot-dev-pass")
	req := httptest.NewRequest(http.MethodPost, "/service/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	AttachCSRFForTest(req)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	if storedFlag == "" {
		t.Fatal("expected ParameterSet to be called")
	}
	if len(storedFlag) < 32 {
		t.Fatalf("expected hashed token flag, got %q", storedFlag)
	}
}

func TestSetupCreatesFirstAccount(t *testing.T) {
	app, _, _ := setupTestAppWithDB(t)
	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "flowbot-dev-pass")
	req := httptest.NewRequest(http.MethodPost, "/service/web/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	AttachCSRFForTest(req)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	if resp.Header.Get("HX-Redirect") != "/service/web/setup/2fa" {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("want setup/2fa redirect, got %q body %s", resp.Header.Get("HX-Redirect"), body)
	}
	ws := store.WebAccountStoreFromDB()
	n, err := ws.Count(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("want 1 account, got n=%d err=%v", n, err)
	}
}

func TestSetupBlockedWhenAccountExists(t *testing.T) {
	app, _, client := setupTestAppWithDB(t)
	seedWebAccount(t, client, "admin", "flowbot-dev-pass", false)
	req := httptest.NewRequest(http.MethodGet, "/service/web/setup", http.NoBody)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("want redirect, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/service/web/login" {
		t.Fatalf("want login redirect, got %q", loc)
	}
}
