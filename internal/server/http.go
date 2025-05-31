package server

import (
	"errors"
	"github.com/bytedance/sonic"
	"github.com/flowline-io/contrib/fiberzerolog"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/samber/oops"
	"time"
)

var sharedApp *fiber.App

func newHTTPServer() *fiber.App {
	// Set up HTTP server.
	app := fiber.New(fiber.Config{
		JSONDecoder:  sonic.Unmarshal,
		JSONEncoder:  sonic.Marshal,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,

		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			// custom error
			var e oops.OopsError
			if errors.As(err, &e) {
				if e.Code() == oops.OopsError(protocol.ErrNotAuthorized).Code() {
					return ctx.Status(fiber.StatusUnauthorized).
						JSON(protocol.NewFailedResponse(e))
				}
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(e))
			}

			// other error
			if err != nil {
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(protocol.ErrBadRequest.Wrap(err)))
			}

			// Return from handler
			return nil
		},
	})
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(requestid.New())
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(limiter.New(limiter.Config{
		Max:               50,
		Expiration:        10 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))
	logger := flog.GetLogger()
	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &logger,
		SkipURIs: []string{
			healthcheck.DefaultLivenessEndpoint,
			healthcheck.DefaultReadinessEndpoint,
			healthcheck.DefaultStartupEndpoint,
			"/",
			"/service/user/metrics",
		},
	}))

	// swagger
	if swagHandler != nil {
		app.Get("/swagger/*", swagHandler)
	}

	// use in registered endpoint
	sharedApp = app

	return app
}
