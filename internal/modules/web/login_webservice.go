package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var loginWebserviceRules = []webservice.Rule{
	webservice.Get("/login", loginPage, route.WithNotAuth()),
	webservice.Post("/login", loginSubmit, route.WithNotAuth()),
	webservice.Post("/logout", logout, route.WithNotAuth()),
	webservice.Get("/csrf-token", csrfTokenJSON, route.WithNotAuth()),
}

const (
	msgAccountLocked         = "Account temporarily locked. Please try again later."
	msgTooManyFailedAttempts = "Too many failed attempts. Account temporarily locked. Please try again later."
	msgInvalidCredentials    = "Invalid username or password"
)

func loginPage(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		next := ctx.Query("next", "/service/web/home")
		return ctx.Redirect().To(next)
	}
	next := ctx.Query("next", "")
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	// Cloudflare and other CDNs must not cache the login HTML or CSRF cookie pairing breaks.
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.LoginPage(next, "", csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

// renderLoginForm writes the login form fragment with a fresh CSRF double-submit token.
func renderLoginForm(ctx fiber.Ctx, next, errorMsg string) error {
	csrfTok, err := ensureCSRFCookie(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "csrf token error")
	}
	ctx.Set("Cache-Control", "no-store")
	ctx.Type("html")
	return pages.LoginForm(next, errorMsg, csrfTok).Render(context.Background(), ctx.Response().BodyWriter())
}

// checkLoginRateLimit checks the rate limiter for the current IP.
// Returns an empty string if the request is allowed, or an error message if blocked.
// When the IP is over the soft threshold, a progressive delay is applied before continuing.
func checkLoginRateLimit(ctx fiber.Ctx) string {
	if loginLimiter == nil {
		return ""
	}
	delay, locked := loginLimiter.Allow(ctx.Context(), ctx.IP())
	if locked {
		return msgAccountLocked
	}
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Context().Done():
			return msgAccountLocked
		case <-timer.C:
		}
	}
	return ""
}

// recordLoginFailure records a failed login attempt and returns the appropriate error message.
func recordLoginFailure(ctx fiber.Ctx) string {
	if loginLimiter == nil {
		return msgInvalidCredentials
	}
	locked, _ := loginLimiter.RecordFailure(ctx.Context(), ctx.IP())
	if locked {
		return msgTooManyFailedAttempts
	}
	return msgInvalidCredentials
}

// loginSuccessCleanup clears rate limit state after a successful login.
func loginSuccessCleanup(ctx fiber.Ctx) {
	if loginLimiter != nil {
		loginLimiter.RecordSuccess(ctx.Context(), ctx.IP())
	}
}

func loginSubmit(ctx fiber.Ctx) error {
	username := ctx.FormValue("username")
	password := ctx.FormValue("password")
	next := ctx.FormValue("next")
	cfg := authConfig()

	if blocked := checkLoginRateLimit(ctx); blocked != "" {
		return renderLoginForm(ctx, next, blocked)
	}

	if username == "" || !cfg.verifyCredentials(username, password) {
		msg := recordLoginFailure(ctx)
		return renderLoginForm(ctx, next, msg)
	}

	loginSuccessCleanup(ctx)

	token, err := auth.NewToken()
	if err != nil {
		flog.Error(fmt.Errorf("failed to generate token: %w", err))
		return renderLoginForm(ctx, next, "Internal error")
	}
	uid := types.Uid("user-" + username)
	params := types.KV{
		"uid":    string(uid),
		"topic":  "web",
		"scopes": []string{"admin:*"},
	}
	expiredAt := time.Now().Add(24 * time.Hour)
	if err := store.Database.ParameterSet(context.Background(), auth.HashToken(token), params, expiredAt); err != nil {
		flog.Error(fmt.Errorf("failed to store token: %w", err))
		return renderLoginForm(ctx, next, "Internal error")
	}
	setAccessTokenCookie(ctx, token, 86400, time.Time{})
	if next == "" || !strings.HasPrefix(next, "/") || strings.Contains(next, "//") || strings.Contains(next, ":") {
		next = "/service/web/home"
	}
	ctx.Set("HX-Redirect", next)
	return nil
}

func logout(ctx fiber.Ctx) error {
	token := ctx.Cookies("accessToken")
	if token != "" {
		if err := route.DeleteAccessToken(context.Background(), token); err != nil {
			flog.Error(fmt.Errorf("failed to delete token on logout: %w", err))
		}
	}
	setAccessTokenCookie(ctx, "deleted", 0, time.Unix(0, 0))
	ctx.Set("HX-Redirect", "/service/web/login")
	return nil
}

// setAccessTokenCookie writes the accessToken cookie with HttpOnly, SameSite=Lax,
// and Secure controlled by modules.web.auth.cookie_secure.
func setAccessTokenCookie(ctx fiber.Ctx, value string, maxAge int, expires time.Time) {
	c := &fiber.Cookie{
		Name:     "accessToken",
		Value:    value,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   authConfig().cookieSecureEnabled(),
		Path:     "/",
		MaxAge:   maxAge,
	}
	if !expires.IsZero() {
		c.Expires = expires
	}
	ctx.Cookie(c)
}
