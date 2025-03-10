package server

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v2"
)

func listenAndServe(app *fiber.App, addr string, stop <-chan bool) error {
	globals.shuttingDown = false

	httpdone := make(chan bool)

	go func() {
		err := app.Listen(addr)
		if err != nil {
			flog.Error(err)
		}
		httpdone <- true
	}()

	// Wait for either a termination signal or an error
Loop:
	for {
		select {
		case <-stop:
			// Flip the flag that we are terminating and close the Accept-ing socket, so no new connections are possible.
			globals.shuttingDown = true
			// Give server 2 seconds to shut down.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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
			cache.Shutdown()

			break Loop
		case <-httpdone:
			break Loop
		}
	}
	return nil
}
