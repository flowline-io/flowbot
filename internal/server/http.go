package server

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/go-playground/validator/v10"
	fiberzerolog "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	tracepkg "github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	defaultRateLimitMax        = 200
	defaultRateLimitExpiration = 10 * time.Second
	// minHTTPWriteTimeout keeps ordinary short responses bounded when RunTimeout is tiny.
	minHTTPWriteTimeout = 90 * time.Second
	// httpWriteTimeoutSlack covers Done flush / title work after the agent turn budget.
	httpWriteTimeoutSlack = time.Minute
)

// httpWriteTimeout returns the Fiber WriteTimeout covering long-lived chatagent SSE streams.
func httpWriteTimeout() time.Duration {
	runTimeout := config.App.ChatAgent.RunTimeout
	if runTimeout <= 0 {
		runTimeout = chatagent.DefaultRunTimeout
	}
	timeout := runTimeout + httpWriteTimeoutSlack
	if timeout < minHTTPWriteTimeout {
		return minHTTPWriteTimeout
	}
	return timeout
}

var (
	sharedApp   *fiber.App
	sharedAppMu sync.RWMutex
)

// sharedAppPtr returns the current shared Fiber app instance.
// Must be called after newHTTPServer has been invoked.
func sharedAppPtr() *fiber.App {
	sharedAppMu.RLock()
	defer sharedAppMu.RUnlock()
	return sharedApp
}

func newHTTPServer() *fiber.App {
	trustedProxies := config.App.HTTP.TrustedProxies
	// Set up HTTP server.
	app := fiber.New(fiber.Config{
		JSONDecoder: sonic.Unmarshal,
		JSONEncoder: sonic.Marshal,
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 30 * time.Second,
		// Chat agent SSE turns can run up to RunTimeout; a fixed 90s WriteTimeout
		// closed chunked responses mid-stream (browser ERR_INCOMPLETE_CHUNKED_ENCODING).
		WriteTimeout: httpWriteTimeout(),
		// Params/headers share fasthttp buffers unless Immutable is set. SSE
		// handlers keep those strings alive after the request ctx is recycled
		// (e.g. concurrent /agents/render-markdown calls), which otherwise
		// corrupts session IDs mid-run.
		Immutable: true,
		// Trust X-Forwarded-For only when trusted_proxies is configured.
		ProxyHeader: fiber.HeaderXForwardedFor,
		TrustProxy:  len(trustedProxies) > 0,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: trustedProxies,
		},

		// validator
		StructValidator: &structValidator{validate: validator.New()},
		// error handler
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			if err == nil {
				return nil
			}
			if status, ok := domainErrorStatus(err); ok {
				flog.Error(err)
				return ctx.Status(status).
					JSON(protocol.NewFailedResponse(err))
			}

			// Fiber errors (e.g. ErrNotFound, ErrMethodNotAllowed)
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				if fiberErr.Code >= 300 && fiberErr.Code < 400 {
					return nil
				}
				return ctx.Status(fiberErr.Code).
					JSON(protocol.NewFailedResponse(err))
			}

			// custom error
			var e oops.OopsError
			if errors.As(err, &e) {
				if e.Code() == protocol.ErrorCode(protocol.ErrNotAuthorized) {
					return ctx.Status(fiber.StatusUnauthorized).
						JSON(protocol.NewFailedResponse(e))
				}
				flog.Error(err)
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(e))
			}

			flog.Error(err)
			return ctx.Status(fiber.StatusInternalServerError).
				JSON(protocol.NewFailedResponse(protocol.ErrInternalServerError.Wrap(err)))
		},
	})
	// recover — log stacks server-side; never rely on client-visible panic detail
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(_ fiber.Ctx, r any) {
			flog.Error(fmt.Errorf("panic recovered: %v\n%s", r, debug.Stack()))
		},
	}))
	// requestid
	app.Use(requestid.New())
	// trace
	app.Use(tracepkg.FiberMiddleware())
	// cors — empty allow_origins does not reflect any Origin (same-origin Web UI).
	// AllowOriginsFunc reads config.App live; AllowCredentials is fixed at startup
	// (Fiber Config is static), so CORS credential mode changes need a process restart.
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return matchHTTPAllowOrigin(config.App.HTTP.CORS.AllowOrigins, origin)
		},
		AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete, fiber.MethodPatch, fiber.MethodOptions},
		AllowHeaders: []string{
			fiber.HeaderOrigin,
			fiber.HeaderContentType,
			fiber.HeaderAccept,
			fiber.HeaderAuthorization,
			"X-AccessToken",
			"X-Request-ID",
		},
		AllowCredentials: corsAllowCredentials(config.App.HTTP.CORS.AllowOrigins),
	}))
	// limiter — static assets and health probes are excluded because each page
	// load issues 10+ /static/* requests that would exhaust the API quota.
	app.Use(limiter.New(limiter.Config{
		Max:               httpRateLimitMax(),
		Expiration:        httpRateLimitExpiration(),
		LimiterMiddleware: limiter.SlidingWindow{},
		Next:              shouldSkipRateLimit,
	}))
	// logger
	app.Use(fiberzerolog.New(fiberzerolog.Config{
		GetLogger: func(_ fiber.Ctx) zerolog.Logger {
			return flog.GetLogger()
		},
		Next: func(c fiber.Ctx) bool {
			skipPaths := []string{
				healthcheck.LivenessEndpoint,
				healthcheck.ReadinessEndpoint,
				healthcheck.StartupEndpoint,
				"/",
				"/metrics",
				"/service/user/metrics",
			}
			return utils.Contains(skipPaths, c.Path())
		},
	}))
	// security headers
	app.Use(securityHeadersMiddleware)

	// static asset caching
	app.Use(staticCacheMiddleware)

	// favicon
	app.Use(favicon.New())

	// swagger
	if swagHandler != nil {
		app.Get("/swagger/*", swagHandler)
	}

	// use in registered endpoint
	sharedAppMu.Lock()
	sharedApp = app
	sharedAppMu.Unlock()

	return app
}

// httpRateLimitMax returns the configured global rate limit max, or the default.
func httpRateLimitMax() int {
	if config.App.HTTP.RateLimit.Max > 0 {
		return config.App.HTTP.RateLimit.Max
	}
	return defaultRateLimitMax
}

// httpRateLimitExpiration returns the configured rate limit window, or the default.
func httpRateLimitExpiration() time.Duration {
	if config.App.HTTP.RateLimit.Expiration > 0 {
		return config.App.HTTP.RateLimit.Expiration
	}
	return defaultRateLimitExpiration
}

// matchHTTPAllowOrigin reports whether origin is permitted by the CORS whitelist.
// An empty whitelist never matches. A sole "*" entry allows any non-empty origin.
func matchHTTPAllowOrigin(allowed []string, origin string) bool {
	if origin == "" || len(allowed) == 0 {
		return false
	}
	if corsAllowsAnyOrigin(allowed) {
		return true
	}
	origin = strings.ToLower(origin)
	for _, val := range allowed {
		if strings.ToLower(val) == origin {
			return true
		}
	}
	return false
}

// corsAllowsAnyOrigin reports whether the whitelist is the open "*" entry.
func corsAllowsAnyOrigin(allowed []string) bool {
	return len(allowed) == 1 && allowed[0] == "*"
}

// corsAllowCredentials reports whether Access-Control-Allow-Credentials should be set.
// Requires a non-empty explicit Origin whitelist; "*" never enables credentials.
func corsAllowCredentials(allowed []string) bool {
	if len(allowed) == 0 || corsAllowsAnyOrigin(allowed) {
		return false
	}
	return true
}

// shouldSkipRateLimit returns true for paths that should not count toward the
// global HTTP rate limiter (static assets, health probes, metrics scraping).
func shouldSkipRateLimit(c fiber.Ctx) bool {
	if strings.HasPrefix(c.Path(), "/static/") {
		return true
	}
	skipPaths := []string{
		healthcheck.LivenessEndpoint,
		healthcheck.ReadinessEndpoint,
		healthcheck.StartupEndpoint,
		"/",
		"/metrics",
		"/service/user/metrics",
	}
	return utils.Contains(skipPaths, c.Path())
}

// securityHeadersMiddleware adds security-related HTTP response headers.
//
// CSP: scripts are self + unsafe-inline (legacy onclick / remaining attribute handlers);
// no 'unsafe-eval' after Tailwind prebuild + Alpine CSP. Prefer removing unsafe-inline next.
func securityHeadersMiddleware(c fiber.Ctx) error {
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderXFrameOptions, "DENY")
	if config.App.ShouldSendHSTS() {
		c.Set(fiber.HeaderStrictTransportSecurity, "max-age=31536000; includeSubDomains")
	}
	if !strings.HasPrefix(c.Path(), "/swagger/") {
		c.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'")
	}
	return c.Next()
}

// staticCacheMiddleware sets long-lived cache headers for static assets.
// Vendor files are version-pinned; app files change less frequently.
func staticCacheMiddleware(c fiber.Ctx) error {
	if !strings.HasPrefix(c.Path(), "/static/") {
		return c.Next()
	}
	if strings.HasPrefix(c.Path(), "/static/vendor/") {
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	} else {
		c.Set(fiber.HeaderCacheControl, "public, max-age=3600")
	}
	return c.Next()
}
