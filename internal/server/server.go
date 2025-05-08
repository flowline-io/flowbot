package server

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/search"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	// bots
	_ "github.com/flowline-io/flowbot/internal/bots/agent"
	_ "github.com/flowline-io/flowbot/internal/bots/anki"
	_ "github.com/flowline-io/flowbot/internal/bots/bookmark"
	_ "github.com/flowline-io/flowbot/internal/bots/clipboard"
	_ "github.com/flowline-io/flowbot/internal/bots/cloudflare"
	_ "github.com/flowline-io/flowbot/internal/bots/dev"
	_ "github.com/flowline-io/flowbot/internal/bots/finance"
	_ "github.com/flowline-io/flowbot/internal/bots/gitea"
	_ "github.com/flowline-io/flowbot/internal/bots/github"
	_ "github.com/flowline-io/flowbot/internal/bots/kanban"
	_ "github.com/flowline-io/flowbot/internal/bots/notify"
	_ "github.com/flowline-io/flowbot/internal/bots/obsidian"
	_ "github.com/flowline-io/flowbot/internal/bots/okr"
	_ "github.com/flowline-io/flowbot/internal/bots/reader"
	_ "github.com/flowline-io/flowbot/internal/bots/search"
	_ "github.com/flowline-io/flowbot/internal/bots/server"
	_ "github.com/flowline-io/flowbot/internal/bots/torrent"
	_ "github.com/flowline-io/flowbot/internal/bots/user"
	_ "github.com/flowline-io/flowbot/internal/bots/webhook"
	_ "github.com/flowline-io/flowbot/internal/bots/workflow"

	// File upload handlers
	_ "github.com/flowline-io/flowbot/pkg/media/fs"
	_ "github.com/flowline-io/flowbot/pkg/media/minio"

	// Notify
	_ "github.com/flowline-io/flowbot/pkg/notify/message-pusher"
	_ "github.com/flowline-io/flowbot/pkg/notify/ntfy"
	_ "github.com/flowline-io/flowbot/pkg/notify/pushover"
	_ "github.com/flowline-io/flowbot/pkg/notify/slack"
)

func RunServer(lc fx.Lifecycle, app *fiber.App, _ *cache.Cache, _ *redis.Client, _ *search.Client) {
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

			// init database
			if err = initializeDatabase(); err != nil {
				return err
			}
			flog.Info("initialize Database ok")

			// init media
			if err = initializeMedia(); err != nil {
				return err
			}
			flog.Info("initialize Media ok")

			// init event
			if err = initializeEvent(); err != nil {
				return err
			}
			flog.Info("initialize Event ok")

			// init chatbot
			if err = initializeChatbot(stopSignal); err != nil {
				return err
			}
			flog.Info("initialize Chatbot ok")

			// init metrics
			if err = initializeMetrics(); err != nil {
				return err
			}
			flog.Info("initialize Metrics ok")

			// http server
			go func() {
				err := app.Listen(config.App.Listen)
				if err != nil {
					flog.Error(err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Give server 2 seconds to shut down.
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			if err := app.ShutdownWithContext(ctx); err != nil {
				// failure/timeout shutting down the server gracefully
				flog.Error(err)
			}

			cancel()

			// Shutdown Extra
			globals.taskQueue.Shutdown()
			globals.manager.Shutdown()
			globals.cronTaskManager.Shutdown()
			for _, ruleset := range globals.cronRuleset {
				ruleset.Shutdown()
			}

			return nil
		},
	})
}
