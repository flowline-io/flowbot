package web

import (
	"context"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/cache"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

// AuthConfig holds web login authentication settings.
// Username/Password/PasswordHash are migration-only after accounts exist in the database.
type AuthConfig struct {
	Username      string           `json:"username"`
	Password      string           `json:"password"`
	PasswordHash  string           `json:"password_hash"`
	EncryptionKey string           `json:"encryption_key"`
	EncryptionDir string           `json:"encryption_key_dir"`
	CookieSecure  *bool            `json:"cookie_secure"`
	BruteForce    BruteForceConfig `json:"brute_force"`
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
func (b *BruteForceConfig) bruteForceEnabled() bool {
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
	wireTOTPRateLimiter()
}

// authConfig returns the parsed authentication configuration.
func authConfig() AuthConfig {
	return handler.authConfig
}

func isAuthenticated(ctx fiber.Ctx) bool {
	if rc := route.GetRequestContext(ctx); rc != nil {
		return webRequestContextOK(ctx, rc)
	}
	token := ctx.Cookies(webauth.CookieAccessToken)
	if token == "" {
		return false
	}
	p, err := route.LookupAccessToken(context.Background(), token)
	if err != nil || p.ID <= 0 || route.AccessTokenIsExpired(p) {
		return false
	}
	paramKV := types.KV(p.Params)
	kind, _ := paramKV.String("kind")
	if kind != webauth.KindFull {
		return false
	}
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

// webRequestContextOK validates a RequestContext already set by route.Authorize.
// Full web sessions require kind=full. API tokens omit kind and use the Authorization
// header; legacy cookie sessions without kind must re-login.
func webRequestContextOK(ctx fiber.Ctx, rc *route.RequestContext) bool {
	kind, _ := rc.Param.String("kind")
	switch kind {
	case webauth.KindFull:
		return true
	case "":
		return ctx.Cookies(webauth.CookieAccessToken) == ""
	default:
		return false
	}
}

func authenticateWeb(ctx fiber.Ctx) error {
	if isAuthenticated(ctx) {
		if uid := getUID(ctx); uid != "" {
			notifypkg.TouchPresence(uid)
		}
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
