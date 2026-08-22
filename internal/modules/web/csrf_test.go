package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
)

func TestCSRFCookieName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		secure *bool
		want   string
	}{
		{name: "HTTPS uses host prefix", secure: new(true), want: csrfCookieNameHost},
		{name: "local HTTP uses csrf_", secure: new(false), want: csrfCookieNameLocal},
		{name: "omitted cookie_secure defaults to host prefix", secure: nil, want: csrfCookieNameHost},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := csrfCookieNameFor(AuthConfig{CookieSecure: tt.secure}); got != tt.want {
				t.Fatalf("csrfCookieNameFor()=%q want %q", got, tt.want)
			}
		})
	}
}

//go:fix inline
func boolPtr(v bool) *bool { return new(v) }

func TestCSRFMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		cookie     string
		header     string
		formToken  string
		session    bool
		wantStatus int
	}{
		{
			name:       "GET does not require CSRF",
			method:     http.MethodGet,
			path:       "/service/web/home",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "unauthenticated logout skips CSRF",
			method:     http.MethodPost,
			path:       "/service/web/logout",
			wantStatus: http.StatusOK,
		},
		{
			name:       "session POST without CSRF rejected",
			method:     http.MethodPost,
			path:       "/service/web/logout",
			session:    true,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "session POST with mismatched CSRF rejected",
			method:     http.MethodPost,
			path:       "/service/web/logout",
			session:    true,
			cookie:     "cookie-token",
			header:     "header-token",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "session POST with matching header accepted",
			method:     http.MethodPost,
			path:       "/service/web/logout",
			session:    true,
			cookie:     "match-token-value-32chars-aaaa",
			header:     "match-token-value-32chars-aaaa",
			wantStatus: http.StatusOK,
		},
		{
			name:       "login POST without CSRF rejected",
			method:     http.MethodPost,
			path:       "/service/web/login",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "login POST with matching form field reaches handler",
			method:     http.MethodPost,
			path:       "/service/web/login",
			cookie:     "form-token-value-32chars-bbbbbb",
			formToken:  "form-token-value-32chars-bbbbbb",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated account password skips CSRF",
			method:     http.MethodPost,
			path:       "/service/web/account/password",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "unauthenticated backup-codes skips CSRF",
			method:     http.MethodPost,
			path:       "/service/web/account/backup-codes",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "session account password without CSRF rejected",
			method:     http.MethodPost,
			path:       "/service/web/account/password",
			session:    true,
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)

			var body *strings.Reader
			if tt.formToken != "" {
				body = strings.NewReader("csrf_token=" + tt.formToken)
			} else {
				body = strings.NewReader("")
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.formToken != "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if tt.session {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			}
			if tt.cookie != "" {
				seedCSRFToken(tt.cookie)
				req.AddCookie(&http.Cookie{Name: csrfCookieName(), Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set(csrfHeaderName, tt.header)
			}
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status=%d want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestEnsureCSRFCookieSetsCookie(t *testing.T) {
	tests := []struct {
		name          string
		existing      string
		seedStore     bool
		wantSetCookie bool
		wantSameValue bool
	}{
		{name: "sets cookie when missing", existing: "", wantSetCookie: true},
		{name: "reuses stored cookie value", existing: "existing-csrf-token-value-xxxxx", seedStore: true, wantSetCookie: true, wantSameValue: true},
		{name: "rotates unknown existing cookie", existing: "short", wantSetCookie: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp(t)
			req := httptest.NewRequest(http.MethodGet, "/service/web/login", http.NoBody)
			if tt.existing != "" {
				if tt.seedStore {
					seedCSRFToken(tt.existing)
				}
				req.AddCookie(&http.Cookie{Name: csrfCookieName(), Value: tt.existing})
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()
			found := ""
			for _, c := range resp.Cookies() {
				if c.Name == csrfCookieName() {
					found = c.Value
				}
			}
			if tt.wantSetCookie && found == "" {
				t.Fatal("want Set-Cookie for CSRF token")
			}
			if tt.wantSameValue && found != tt.existing {
				t.Fatalf("cookie=%q want %q", found, tt.existing)
			}
			if tt.existing == "short" && found != "" && found == tt.existing {
				t.Fatal("want rotated csrf cookie for unknown token")
			}
		})
	}
}

func TestCSRFCookieFlags(t *testing.T) {
	tests := []struct {
		name       string
		secure     bool
		wantName   string
		wantSecure bool
	}{
		{name: "local HTTP cookie is csrf_ without Secure", secure: false, wantName: csrfCookieNameLocal, wantSecure: false},
		{name: "HTTPS cookie is __Host-csrf_ with Secure", secure: true, wantName: csrfCookieNameHost, wantSecure: true},
		{name: "session cookie omits Max-Age", secure: false, wantName: csrfCookieNameLocal, wantSecure: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			lockWebTestGlobals(t)
			secure := tt.secure
			handler = moduleHandler{authConfig: AuthConfig{CookieSecure: &secure}}
			config = configType{Enabled: true, Auth: AuthConfig{CookieSecure: &secure}}

			app := fiber.New()
			app.Use("/service/web", newCSRFMiddleware())
			app.Get("/service/web/login", func(c fiber.Ctx) error {
				return c.SendString("ok")
			})
			req := httptest.NewRequest(http.MethodGet, "/service/web/login", http.NoBody)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			var found *http.Cookie
			for _, c := range resp.Cookies() {
				if c.Name == tt.wantName {
					found = c
					break
				}
			}
			if found == nil {
				t.Fatalf("want Set-Cookie %q", tt.wantName)
			}
			if found.Secure != tt.wantSecure {
				t.Fatalf("Secure=%v want %v", found.Secure, tt.wantSecure)
			}
			if found.HttpOnly {
				t.Fatal("HttpOnly must be false so JS can read the token")
			}
			if found.Path != "/" {
				t.Fatalf("Path=%q want /", found.Path)
			}
			if found.SameSite != http.SameSiteLaxMode {
				t.Fatalf("SameSite=%v want Lax", found.SameSite)
			}
			if found.MaxAge != 0 {
				t.Fatalf("MaxAge=%d want 0 (session)", found.MaxAge)
			}
		})
	}
}

func TestAttachCSRFForTest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "sets cookie and header"},
		{name: "idempotent second call"},
		{name: "token non-empty"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/service/web/logout", http.NoBody)
			AttachCSRFForTest(req)
			if tt.name == "idempotent second call" {
				AttachCSRFForTest(req)
			}
			c, err := req.Cookie(csrfCookieNameLocal)
			if err != nil {
				c, err = req.Cookie(csrfCookieNameHost)
			}
			if err != nil || c.Value == "" {
				t.Fatalf("cookie: %v value=%q", err, c)
			}
			if req.Header.Get(csrfHeaderName) == "" {
				t.Fatal("want CSRF header")
			}
			if req.Header.Get(csrfHeaderName) != c.Value {
				t.Fatal("header must match cookie")
			}
		})
	}
}

func TestRequestPublicOrigin(t *testing.T) {
	tests := []struct {
		name       string
		configURL  string
		host       string
		proto      string
		wantOrigin string
	}{
		{name: "prefers config url", configURL: "https://bot.example/", host: "localhost", wantOrigin: "https://bot.example"},
		{name: "http from request", configURL: "", host: "flowbot.local:8080", wantOrigin: "http://flowbot.local:8080"},
		{name: "https from forwarded proto", configURL: "", host: "bot.example", proto: "https", wantOrigin: "https://bot.example"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			prev := pkgconfig.App.Flowbot.URL
			pkgconfig.App.Flowbot.URL = tt.configURL
			t.Cleanup(func() { pkgconfig.App.Flowbot.URL = prev })

			app := fiber.New()
			var got string
			app.Get("/t", func(c fiber.Ctx) error {
				got = requestPublicOrigin(c)
				return nil
			})
			req := httptest.NewRequest(http.MethodGet, "http://"+tt.host+"/t", http.NoBody)
			req.Host = tt.host
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()
			if got != tt.wantOrigin {
				t.Fatalf("origin=%q want %q", got, tt.wantOrigin)
			}
		})
	}
}
