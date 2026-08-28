package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/i18n"
)

func TestLocaleSwitchSetsCookie(t *testing.T) {
	t.Parallel()
	handler.initialized = true
	app := fiber.New()
	app.Post("/service/web/locale", localeSwitch)

	req := httptest.NewRequest(http.MethodPost, "/service/web/locale", strings.NewReader("lang=zh"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d want 204", resp.StatusCode)
	}
	var langCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == i18n.CookieName {
			langCookie = c
			break
		}
	}
	if langCookie == nil || langCookie.Value != i18n.CookieZH {
		t.Fatalf("cookie: got %+v want flowbot-lang=zh", langCookie)
	}
	if resp.Header.Get("HX-Refresh") != "true" {
		t.Fatalf("want HX-Refresh header")
	}
	_, _ = io.Copy(io.Discard, resp.Body)
}

func TestLocaleMiddlewareDefaultsEnglish(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Use(localeMiddleware())
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString(i18n.T(c.Context(), "nav.inbox"))
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Inbox" {
		t.Fatalf("body: got %q want Inbox", string(body))
	}
}

func TestLocaleMiddlewareChineseCookie(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Use(localeMiddleware())
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString(i18n.T(c.Context(), "nav.inbox"))
	})
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: i18n.CookieName, Value: i18n.CookieZH})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "收件箱" {
		t.Fatalf("body: got %q want 收件箱", string(body))
	}
}

func TestLocaleMiddlewareStoresCookieLang(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Use(localeMiddleware())
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString(i18n.CookieLang(c.Context()))
	})
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: i18n.CookieName, Value: i18n.CookieZH})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != i18n.CookieZH {
		t.Fatalf("body: got %q want %s", string(body), i18n.CookieZH)
	}
}
