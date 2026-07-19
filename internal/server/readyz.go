package server

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/rdb"
)

// httpStopping is set true when the HTTP server begins graceful shutdown so
// /readyz fails before the listener closes.
var httpStopping atomic.Bool

// setHTTPStopping marks the process as no longer ready for new traffic.
func setHTTPStopping(v bool) {
	httpStopping.Store(v)
}

// isHTTPStopping reports whether readiness should fail for shutdown drain.
func isHTTPStopping() bool {
	return httpStopping.Load()
}

// readinessOK reports whether Postgres and Redis respond to ping and the
// server is not shutting down.
func readinessOK(ctx context.Context) bool {
	if isHTTPStopping() {
		return false
	}
	if store.Database == nil || rdb.Client == nil {
		return false
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := store.Database.Ping(pingCtx); err != nil {
		return false
	}
	if err := rdb.Client.Ping(pingCtx).Err(); err != nil {
		return false
	}
	return true
}

// readinessHandler is a Fiber readiness probe used by /readyz.
func readinessHandler(c fiber.Ctx) error {
	if readinessOK(c.Context()) {
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendStatus(fiber.StatusServiceUnavailable)
}
