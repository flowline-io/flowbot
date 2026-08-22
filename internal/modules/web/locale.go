package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/i18n"
)

const localeCookieMaxAge = 365 * 24 * time.Hour

func localeMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		cookie := i18n.ParseCookie(string(c.Cookies(i18n.CookieName)))
		loc := i18n.LocalizerForCookie(cookie)
		c.SetContext(i18n.WithLocalizer(c.Context(), loc))
		return c.Next()
	}
}

func localeSwitch(ctx fiber.Ctx) error {
	lang := strings.TrimSpace(ctx.FormValue("lang"))
	if lang == "" {
		lang = strings.TrimSpace(ctx.Query("lang"))
	}
	lang = i18n.ParseCookie(lang)
	ctx.Cookie(&fiber.Cookie{
		Name:     i18n.CookieName,
		Value:    lang,
		Path:     "/",
		MaxAge:   int(localeCookieMaxAge.Seconds()),
		SameSite: "Lax",
		HTTPOnly: false,
		Secure:   config.Auth.cookieSecureEnabled(),
	})
	ctx.Set("HX-Refresh", "true")
	return ctx.SendStatus(fiber.StatusNoContent)
}

// AttachLocaleForTest sets flowbot-lang when the request has no locale cookie yet.
// Use en for stable English assertions; zh for localized smoke tests.
func AttachLocaleForTest(req *http.Request, lang string) {
	if req == nil {
		return
	}
	lang = i18n.ParseCookie(lang)
	existing := req.Header.Get("Cookie")
	if strings.Contains(existing, i18n.CookieName+"=") {
		return
	}
	req.AddCookie(&http.Cookie{Name: i18n.CookieName, Value: lang})
}
