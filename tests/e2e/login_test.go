//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginPage(t *testing.T) {
	tests := []struct {
		name     string
		wantText string
	}{
		{"renders login form", "Flowbot"},
		{"has username field", "Username"},
		{"has login button", "Login"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := NewPage(t)
			page.MustNavigate(URL("/login"))
			page.MustWaitStable()
			body := page.MustElement("body").MustText()
			assert.Contains(t, body, tt.wantText)
		})
	}
}

func TestLoginSubmit(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{"valid credentials succeeds", "admin", "e2e-test-pass", false},
		{"invalid username shows error", "wrong", "e2e-test-pass", true},
		{"empty credentials shows error", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := NewPage(t)
			page.MustNavigate(URL("/login"))
			page.MustWaitStable()

			page.MustElement(`[data-testid="login-username"]`).MustInput(tt.username)
			page.MustElement(`[data-testid="login-password"]`).MustInput(tt.password)
			wait := page.MustWaitRequestIdle()
			page.MustElement(`[data-testid="login-submit"]`).MustClick()
			wait()

			if tt.wantErr {
				page.MustElement(`[data-testid="login-error"]`)
			} else {
				page.MustWaitStable()
				cookies := page.MustCookies(baseURL)
				var hasAccessToken bool
				for _, c := range cookies {
					if c.Name == "accessToken" {
						hasAccessToken = true
						break
					}
				}
				assert.True(t, hasAccessToken, "accessToken cookie should be set")
			}
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"logout clears cookie and redirects"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := loginViaCookie(t)

			wait := page.MustWaitRequestIdle()
			page.MustNavigate(URL("/configs"))
			wait()
			page.MustWaitStable()

			wait = page.MustWaitRequestIdle()
			page.MustElement(`[data-testid="nav-logout"]`).MustClick()
			wait()

			page.MustWaitStable()

			txt := page.MustElement("body").MustText()
			assert.Contains(t, txt, "Flowbot")

			cookies := page.MustCookies(baseURL)
			for _, c := range cookies {
				assert.NotEqual(t, "accessToken", c.Name, "accessToken cookie should be cleared")
			}
		})
	}
}
