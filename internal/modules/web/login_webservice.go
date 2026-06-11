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
}

const (
	msgAccountLocked         = "Account temporarily locked. Please try again later."
	msgTooManyFailedAttempts = "Too many failed attempts. Account temporarily locked. Please try again later."
	msgInvalidCredentials    = "Invalid username or password"
)

func loginPage(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		next := ctx.Query("next", "/service/web/configs")
		return ctx.Redirect().To(next)
	}
	next := ctx.Query("next", "")
	ctx.Type("html")
	return pages.LoginPage(next, "").Render(context.Background(), ctx.Response().BodyWriter())
}

// checkLoginRateLimit checks the rate limiter for the current IP.
// Returns an empty string if the request is allowed, or an error message if blocked.
func checkLoginRateLimit(ctx fiber.Ctx) string {
	if loginLimiter == nil {
		return ""
	}
	_, locked := loginLimiter.Allow(ctx.Context(), ctx.IP())
	if locked {
		return msgAccountLocked
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
		ctx.Type("html")
		return pages.LoginForm(next, blocked).Render(context.Background(), ctx.Response().BodyWriter())
	}

	if username == "" || username != cfg.Username || password != cfg.Password {
		msg := recordLoginFailure(ctx)
		ctx.Type("html")
		return pages.LoginForm(next, msg).Render(context.Background(), ctx.Response().BodyWriter())
	}

	loginSuccessCleanup(ctx)

	token, err := auth.NewToken()
	if err != nil {
		flog.Error(fmt.Errorf("failed to generate token: %w", err))
		ctx.Type("html")
		return pages.LoginForm(next, "Internal error").Render(context.Background(), ctx.Response().BodyWriter())
	}
	uid := types.Uid("user-" + username)
	params := types.KV{
		"uid":    string(uid),
		"topic":  "web",
		"scopes": []string{"admin:*"},
	}
	expiredAt := time.Now().Add(24 * time.Hour)
	if err := store.Database.ParameterSet(context.Background(), token, params, expiredAt); err != nil {
		flog.Error(fmt.Errorf("failed to store token: %w", err))
		ctx.Type("html")
		return pages.LoginForm(next, "Internal error").Render(context.Background(), ctx.Response().BodyWriter())
	}
	ctx.Cookie(&fiber.Cookie{
		Name:     "accessToken",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   86400,
	})
	if next == "" || !strings.HasPrefix(next, "/") || strings.Contains(next, "//") || strings.Contains(next, ":") {
		next = "/service/web/configs"
	}
	ctx.Set("HX-Redirect", next)
	return nil
}

func logout(ctx fiber.Ctx) error {
	token := ctx.Cookies("accessToken")
	if token != "" {
		if err := store.Database.ParameterDelete(context.Background(), token); err != nil {
			flog.Error(fmt.Errorf("failed to delete token on logout: %w", err))
		}
	}
	ctx.Cookie(&fiber.Cookie{
		Name:     "accessToken",
		Value:    "deleted",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
	})
	ctx.Set("HX-Redirect", "/service/web/login")
	return nil
}
