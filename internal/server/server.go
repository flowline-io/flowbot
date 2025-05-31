package server

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/search"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func RunServer(lc fx.Lifecycle, app *fiber.App, _ store.Adapter, _ *cache.Cache, _ *redis.Client, _ *search.Client, _ message.Publisher,
	_ *workflow.Queue, _ *workflow.Manager, _ *workflow.CronTaskManager) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var err error

			// init log
			if err = initializeLog(); err != nil {
				return err
			}
			flog.Info("initialize Log ok")

			// init timezone
			if err = initializeTimezone(); err != nil {
				return err
			}
			flog.Info("initialize Timezone ok")

			// init media
			if err = initializeMedia(); err != nil {
				return err
			}
			flog.Info("initialize Media ok")

			// init metrics
			if err = initializeMetrics(); err != nil {
				return err
			}
			flog.Info("initialize Metrics ok")

			// init rule engine
			if err = initializeRuleEngine(app); err != nil {
				return err
			}
			flog.Info("initialize Rule Engine ok")

			// http server
			go func() {
				err := app.Listen(config.App.Listen, fiber.ListenConfig{
					DisableStartupMessage: true,
					EnablePrintRoutes:     true,
				})
				if err != nil {
					flog.Error(err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Give server 10 seconds to shut down.
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			if err := app.ShutdownWithContext(ctx); err != nil {
				// failure/timeout shutting down the server gracefully
				flog.Error(err)
			}

			cancel()

			// Shutdown Extra
			for _, ruleset := range globals.cronRuleset {
				ruleset.Shutdown()
			}

			return nil
		},
	})
}
