package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
)

// bindHTTPServer opens the HTTP listener before serving so port conflicts fail
// fx startup synchronously instead of racing inside a background goroutine.
func bindHTTPServer(addr string) (net.Listener, error) {
	if utils.IsUnixAddr(addr) {
		return utils.NetListener(addr)
	}
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}
	return ln, nil
}

// shouldIgnoreServeError reports whether a Fiber listener exit is expected during shutdown.
func shouldIgnoreServeError(err error, stopping *atomic.Bool) bool {
	if err == nil {
		return true
	}
	if stopping != nil && stopping.Load() {
		return true
	}
	return errors.Is(err, http.ErrServerClosed)
}

// serveFiberListener runs app.Listener in the background. Unexpected exit triggers fx
// shutdown so the process does not keep running without a listening HTTP port.
func serveFiberListener(app *fiber.App, ln net.Listener, shutdowner fx.Shutdowner, stopping *atomic.Bool) {
	go func() {
		flog.Info("start http server, listen on %s", ln.Addr().String())
		// Do not pass fx OnStart context to GracefulContext: it is cancelled when
		// startup hooks finish, which would stop the HTTP server shortly after boot.
		serveErr := app.Listener(ln, fiber.ListenConfig{
			DisableStartupMessage: true,
			EnablePrintRoutes:     true,
		})
		if shouldIgnoreServeError(serveErr, stopping) {
			return
		}
		flog.Error(fmt.Errorf("http server stopped unexpectedly: %w", serveErr))
		if err := shutdowner.Shutdown(fx.ExitCode(1)); err != nil {
			flog.Error(fmt.Errorf("fx shutdown after http server exit: %w", err))
		}
	}()
}

func RunServer(
	lc fx.Lifecycle,
	app *fiber.App,
	shutdowner fx.Shutdowner,
	_ store.Adapter,
	_ *cache.Cache,
	_ *redis.Client,
	_ message.Publisher,
) {
	var stopping atomic.Bool

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			var err error

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

			ln, err := bindHTTPServer(config.App.Listen)
			if err != nil {
				return err
			}

			serveFiberListener(app, ln, shutdowner, &stopping)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopping.Store(true)

			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := app.ShutdownWithContext(ctx); err != nil {
				flog.Error(err)
			}

			capability.ShutdownEventPool()

			return nil
		},
	})
}
