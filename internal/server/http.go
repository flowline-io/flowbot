package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/version"
	"github.com/gofiber/fiber/v2"
	json "github.com/json-iterator/go"
)

func listenAndServe(app *fiber.App, addr string, tlfConf *tls.Config, stop <-chan bool) error {
	globals.shuttingDown = false

	httpdone := make(chan bool)

	go func() {
		if tlfConf != nil {
			err := app.ListenTLSWithCertificate(addr, tlfConf.Certificates[0])
			if err != nil {
				flog.Error(err)
			}
		} else {
			err := app.Listen(addr)
			if err != nil {
				flog.Error(err)
			}
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

			// Stop publishing statistics.
			stats.Shutdown()

			cancel()

			// Shutdown Extra
			globals.crawler.Shutdown()
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

func signalHandler() <-chan bool {
	stop := make(chan bool)

	signchan := make(chan os.Signal, 1)
	signal.Notify(signchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		// Wait for a signal. Don't care which signal it is
		sig := <-signchan
		flog.Info("Signal received: '%s', shutting down", sig)
		stop <- true
	}()

	return stop
}

// debugDump is server internal state dump for debugging.
type debugDump struct {
	Version   string    `json:"server_version,omitempty"`
	Build     string    `json:"build_id,omitempty"`
	Timestamp time.Time `json:"ts,omitempty"`
}

func serveStatus(wrt http.ResponseWriter, _ *http.Request) {
	wrt.Header().Set("Content-Type", "application/json")

	result := &debugDump{
		Version:   version.Buildtags,
		Build:     version.Buildstamp,
		Timestamp: types.TimeNow(),
	}

	_ = json.NewEncoder(wrt).Encode(result)
}
