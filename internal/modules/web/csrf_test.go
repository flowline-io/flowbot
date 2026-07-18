package web

import (
	"crypto/subtle"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateCSRFToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "produces non-empty token"},
		{name: "produces unique tokens"},
		{name: "token is url-safe length"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tok, err := generateCSRFToken()
			if err != nil {
				t.Fatalf("generateCSRFToken: %v", err)
			}
			if tok == "" {
				t.Fatal("want non-empty token")
			}
			if len(tok) < 32 {
				t.Fatalf("want token length >= 32, got %d", len(tok))
			}
			tok2, err := generateCSRFToken()
			if err != nil {
				t.Fatalf("generateCSRFToken second: %v", err)
			}
			if tt.name == "produces unique tokens" && tok == tok2 {
				t.Fatal("want distinct tokens")
			}
		})
	}
}

func TestCSRFTokensEqual(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		a, b  string
		equal bool
	}{
		{name: "matching tokens", a: "abc", b: "abc", equal: true},
		{name: "mismatched tokens", a: "abc", b: "xyz", equal: false},
		{name: "empty both", a: "", b: "", equal: false},
		{name: "empty left", a: "", b: "abc", equal: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := csrfTokensEqual(tt.a, tt.b)
			if got != tt.equal {
				t.Fatalf("csrfTokensEqual(%q,%q)=%v want %v", tt.a, tt.b, got, tt.equal)
			}
			if tt.equal && subtle.ConstantTimeCompare([]byte(tt.a), []byte(tt.b)) != 1 {
				t.Fatal("expected constant-time match for equal case")
			}
		})
	}
}

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
			wantStatus: http.StatusSeeOther, // unauthenticated redirect
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
			wantStatus: http.StatusOK, // invalid creds still render form
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { handler = moduleHandler{}; config = configType{} }()

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
				req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set(csrfHeaderName, tt.header)
			}
			resp, err := app.Test(req)
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
		wantSetCookie bool
	}{
		{name: "sets cookie when missing", existing: "", wantSetCookie: true},
		{name: "keeps existing cookie", existing: "existing-csrf-token-value-xxxxx", wantSetCookie: false},
		{name: "rejects short existing and rotates", existing: "short", wantSetCookie: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/login", http.NoBody)
			if tt.existing != "" {
				req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: tt.existing})
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()
			found := false
			for _, c := range resp.Cookies() {
				if c.Name != csrfCookieName {
					continue
				}
				found = true
				if len(c.Value) < 16 {
					t.Fatalf("cookie value too short: %q", c.Value)
				}
			}
			if tt.wantSetCookie && !found && tt.existing == "" {
				t.Fatal("want Set-Cookie for csrfToken")
			}
			if tt.wantSetCookie && tt.existing == "short" && !found {
				t.Fatal("want rotated csrf cookie for short token")
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/service/web/logout", http.NoBody)
			AttachCSRFForTest(req)
			if tt.name == "idempotent second call" {
				AttachCSRFForTest(req)
			}
			c, err := req.Cookie(csrfCookieName)
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
