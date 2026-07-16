package web

import (
	"context"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

// AuthConfig holds web login authentication credentials read from the module config.
type AuthConfig struct {
	Username   string           `json:"username"`
	Password   string           `json:"password"`
	BruteForce BruteForceConfig `json:"brute_force"`
}

// BruteForceConfig holds brute force protection settings for the login endpoint.
type BruteForceConfig struct {
	Enabled         bool   `json:"enabled"`
	MaxAttempts     int64  `json:"max_attempts"`
	LockoutAttempts int64  `json:"lockout_attempts"`
	LockoutDuration string `json:"lockout_duration"`
	WindowDuration  string `json:"window_duration"`
}

// loginLimiter is the rate limiter instance, set during Init if brute force is enabled.
var loginLimiter *loginRateLimiter

// SetLoginRateLimiterCache sets the cache backend for the login rate limiter.
// Must be called after Init if BruteForce is enabled.
func SetLoginRateLimiterCache(s *cache.RedisStore) {
	if config.Auth.BruteForce.Enabled {
		lockoutTTL, err := time.ParseDuration(config.Auth.BruteForce.LockoutDuration)
		if err != nil || lockoutTTL <= 0 {
			lockoutTTL = 15 * time.Minute
		}
		windowTTL, err := time.ParseDuration(config.Auth.BruteForce.WindowDuration)
		if err != nil || windowTTL <= 0 {
			windowTTL = 15 * time.Minute
		}
		maxAttempts := config.Auth.BruteForce.MaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = 5
		}
		lockoutLimit := config.Auth.BruteForce.LockoutAttempts
		if lockoutLimit <= 0 {
			lockoutLimit = 10
		}
		loginLimiter = newLoginRateLimiter(s, maxAttempts, lockoutLimit, cache.TTL(windowTTL), cache.TTL(lockoutTTL))
	}
}

// authConfig returns the parsed authentication configuration.
func authConfig() AuthConfig {
	return handler.authConfig
}

func isAuthenticated(ctx fiber.Ctx) bool {
	if route.GetRequestContext(ctx) != nil {
		return true
	}
	token := ctx.Cookies("accessToken")
	if token == "" {
		return false
	}
	p, err := route.LookupAccessToken(context.Background(), token)
	if err != nil || p.ID <= 0 || store.ParameterIsExpired(p) {
		return false
	}
	paramKV := types.KV(p.Params)
	uidStr, _ := paramKV.String("uid")
	uid := types.Uid(uidStr)
	if uid.IsZero() {
		return false
	}
	topic, _ := paramKV.String("topic")
	var scopes []string
	if raw, ok := paramKV["scopes"]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					scopes = append(scopes, s)
				}
			}
		case []string:
			scopes = v
		}
	}
	ctx.Locals("route:ctx", &route.RequestContext{
		UID:    uid,
		Topic:  topic,
		Param:  paramKV,
		Scopes: scopes,
	})
	return true
}

func authenticateWeb(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		return nil
	}
	return redirectToLogin(ctx)
}

func redirectToLogin(ctx fiber.Ctx) error {
	next := string(ctx.Request().URI().RequestURI())
	nextEncoded := url.QueryEscape(next)
	ctx.Redirect().To("/service/web/login?next=" + nextEncoded)
	return fiber.NewError(fiber.StatusSeeOther, "redirect to login")
}
