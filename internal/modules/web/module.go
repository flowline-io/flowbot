// Package web provides a web UI module with server-rendered HTML pages.
package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	webassets "github.com/flowline-io/flowbot"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
)

const Name = "web"

var handler moduleHandler
var config configType

// Register registers the web module handler.
func Register() {
	module.Register(Name, &handler)
}

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

type moduleHandler struct {
	initialized bool
	authConfig  AuthConfig
	module.Base
}

type configType struct {
	Enabled bool       `json:"enabled"`
	Auth    AuthConfig `json:"auth"`
}

// loginLimiter is the rate limiter instance, set during Init if brute force is enabled.
var loginLimiter *loginRateLimiter

// SetLoginRateLimiterCache sets the cache backend for the login rate limiter.
// Must be called after Init if BruteForce is enabled.
func SetLoginRateLimiterCache(store rateLimitStore) {
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
		loginLimiter = newLoginRateLimiter(store, maxAttempts, lockoutLimit, cache.TTL(windowTTL), cache.TTL(lockoutTTL))
	}
}

// Init initializes the web module with the given JSON configuration.
func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	handler.initialized = true
	handler.authConfig = config.Auth
	return nil
}

// IsReady checks if the web module is initialized.
func (moduleHandler) IsReady() bool {
	return handler.initialized
}

// Bootstrap performs post-initialization setup.
func (moduleHandler) Bootstrap() error {
	return nil
}

// Webservice mounts web module routes on the fiber app.
func (moduleHandler) Webservice(app *fiber.App) {
	app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))
	module.Webservice(app, Name, webserviceRules)
	module.Webservice(app, Name, pipelineWebserviceRules)
	module.Webservice(app, Name, viewWebserviceRules)
}

// Rules returns the web module rule definitions.
func (moduleHandler) Rules() []any {
	return []any{webserviceRules, pipelineWebserviceRules, viewWebserviceRules}
}

// InitForE2E initializes the web module handler for e2e testing.
// It calls the package-level handler.Init with the provided JSON config,
// bypassing the uber/fx dependency injection used in production.
func InitForE2E(configData json.RawMessage) error {
	return handler.Init(configData)
}

// MountForE2E mounts web module routes onto the given Fiber app.
func MountForE2E(app *fiber.App) {
	handler.Webservice(app)
}

// authConfig returns the parsed authentication configuration.
func authConfig() AuthConfig {
	return handler.authConfig
}
