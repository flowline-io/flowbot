package server

import (
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	jsoniter "github.com/json-iterator/go"
)

func newHTTPServer() *fiber.App {
	// Set up HTTP server.
	httpApp := fiber.New(fiber.Config{
		DisableStartupMessage: true,

		JSONDecoder:  jsoniter.Unmarshal,
		JSONEncoder:  jsoniter.Marshal,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,

		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			// Send custom error page
			if err != nil {
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponseWithError(protocol.ErrBadRequest, err))
			}

			// Return from handler
			return nil
		},
	})
	httpApp.Use(recover.New(recover.Config{EnableStackTrace: true}))
	httpApp.Use(requestid.New())
	httpApp.Use(healthcheck.New())
	httpApp.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
	}))
	httpApp.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	httpApp.Use(limiter.New(limiter.Config{
		Max:               50,
		Expiration:        10 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))
	logger := flog.GetLogger()
	httpApp.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &logger,
		SkipURIs: []string{
			"/",
			"/livez",
			"/readyz",
			"/service/user/metrics",
		},
	}))

	// hook
	httpApp.Hooks().OnRoute(func(r fiber.Route) error {
		if r.Method == http.MethodHead {
			return nil
		}
		flog.Info("[route] %+7s %s", r.Method, r.Path)
		return nil
	})

	// swagger
	if swagHandler != nil {
		httpApp.Get("/swagger/*", swagHandler)
	}

	return httpApp
}
