package server

import (
	"errors"
	"fmt"
	"runtime/trace"
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
	"github.com/gofiber/fiber/v3/middleware/pprof"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/samber/oops"

	"github.com/flowline-io/flowbot/pkg/flog"
	tracepkg "github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

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
	// Set up HTTP server.
	app := fiber.New(fiber.Config{
		JSONDecoder:  sonic.Unmarshal,
		JSONEncoder:  sonic.Marshal,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,

		// validator
		StructValidator: &structValidator{validate: validator.New()},
		// error handler
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			if err == nil {
				return nil
			}
			if status, ok := domainErrorStatus(err); ok {
				return ctx.Status(status).
					JSON(protocol.NewFailedResponse(err))
			}

			// Fiber errors (e.g. ErrNotFound, ErrMethodNotAllowed)
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
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
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(e))
			}

			return ctx.Status(fiber.StatusInternalServerError).
				JSON(protocol.NewFailedResponse(protocol.ErrInternalServerError.Wrap(err)))
		},
	})
	// recover
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	// requestid
	app.Use(requestid.New())
	// trace
	app.Use(tracepkg.FiberMiddleware())
	// cors
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(_ string) bool {
			return true
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
		AllowCredentials: true,
	}))
	// limiter
	app.Use(limiter.New(limiter.Config{
		Max:               50,
		Expiration:        10 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
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
				"/service/user/metrics",
			}
			return lo.Contains(skipPaths, c.Path())
		},
	}))
	// pprof
	app.Use(pprof.New(pprof.Config{Prefix: "/server-debugger", Next: authPprof}))

	// flight recorder
	fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{})
	if err := fr.Start(); err != nil {
		flog.Error(fmt.Errorf("failed to start flight recorder: %w", err))
	} else {
		flog.Info("flight recorder started")

		// add debug route for flight recorder
		app.Get("/server-debugger/debug/trace", func(c fiber.Ctx) error {
			// Use the same auth logic as pprof
			if authPprof(c) {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			c.Set("Content-Type", "application/octet-stream")
			c.Set("Content-Disposition", `attachment; filename="trace.out"`)

			_, err := fr.WriteTo(c.Response().BodyWriter())
			return err
		})
	}

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
