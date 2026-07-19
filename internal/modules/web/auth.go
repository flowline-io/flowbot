package web

import (
	"context"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

// AuthConfig holds web login authentication credentials read from the module config.
// Configure either Password (development plaintext) or PasswordHash (bcrypt, production).
type AuthConfig struct {
	Username     string           `json:"username"`
	Password     string           `json:"password"`
	PasswordHash string           `json:"password_hash"`
	CookieSecure *bool            `json:"cookie_secure"`
	BruteForce   BruteForceConfig `json:"brute_force"`
}

// cookieSecureEnabled reports whether the accessToken cookie should set Secure.
// Defaults to true when cookie_secure is omitted (HTTPS / frp deployments).
func (a AuthConfig) cookieSecureEnabled() bool {
	if a.CookieSecure == nil {
		return true
	}
	return *a.CookieSecure
}

// BruteForceConfig holds brute force protection settings for the login endpoint.
type BruteForceConfig struct {
	// Enabled turns protection on when true or omitted (nil). Set false to disable.
	Enabled *bool `json:"enabled"`
	// MaxAttempts is when progressive delay starts (0 = default 5).
	MaxAttempts int64 `json:"max_attempts"`
	// LockoutAttempts is when full lockout starts (0 = default 10).
	LockoutAttempts int64 `json:"lockout_attempts"`
	// LockoutDuration is how long lockout lasts (empty = default 15m).
	LockoutDuration string `json:"lockout_duration"`
	// WindowDuration is the sliding window for attempt counts (empty = default 15m).
	WindowDuration string `json:"window_duration"`
}

// bruteForceEnabled reports whether login brute-force protection is active.
// Defaults to true when enabled is omitted.
func (b BruteForceConfig) bruteForceEnabled() bool {
	if b.Enabled == nil {
		return true
	}
	return *b.Enabled
}

// applyDefaults fills zero BruteForce numeric/duration fields with built-in defaults.
func (b *BruteForceConfig) applyDefaults() {
	if b.MaxAttempts <= 0 {
		b.MaxAttempts = 5
	}
	if b.LockoutAttempts <= 0 {
		b.LockoutAttempts = 10
	}
	if b.LockoutDuration == "" {
		b.LockoutDuration = "15m"
	}
	if b.WindowDuration == "" {
		b.WindowDuration = "15m"
	}
}

// loginLimiter is the rate limiter instance, set after Init when brute force is enabled.
var loginLimiter *loginRateLimiter

// loginLimiterStore is the Redis cache injected via fx; limiter wiring waits until Init.
var loginLimiterStore *cache.RedisStore

// SetLoginRateLimiterCache stores the Redis backend for the login rate limiter.
// The limiter is wired after web module Init so YAML auth.brute_force is applied.
func SetLoginRateLimiterCache(s *cache.RedisStore) {
	loginLimiterStore = s
	wireLoginRateLimiter()
}

// wireLoginRateLimiter builds or clears loginLimiter from the current module auth config.
// No-op until Init has succeeded and a Redis store is available.
func wireLoginRateLimiter() {
	if loginLimiterStore == nil || !handler.initialized {
		return
	}
	if !config.Auth.BruteForce.bruteForceEnabled() {
		loginLimiter = nil
		return
	}
	bf := config.Auth.BruteForce
	bf.applyDefaults()
	lockoutTTL, err := time.ParseDuration(bf.LockoutDuration)
	if err != nil || lockoutTTL <= 0 {
		lockoutTTL = 15 * time.Minute
	}
	windowTTL, err := time.ParseDuration(bf.WindowDuration)
	if err != nil || windowTTL <= 0 {
		windowTTL = 15 * time.Minute
	}
	loginLimiter = newLoginRateLimiter(loginLimiterStore, bf.MaxAttempts, bf.LockoutAttempts, cache.TTL(windowTTL), cache.TTL(lockoutTTL))
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
	if !auth.HasAnyScope(scopes) {
		return false
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
