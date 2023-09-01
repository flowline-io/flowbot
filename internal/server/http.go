package server

import (
	"context"
	"crypto/tls"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/flowline-io/flowbot/pkg/queue"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/version"
	"github.com/gofiber/fiber/v2"
	json "github.com/json-iterator/go"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

func listenAndServe(app *fiber.App, addr string, tlfConf *tls.Config, stop <-chan bool) error {
	globals.shuttingDown = false

	httpdone := make(chan bool)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(map[string]any{"flowbot": currentVersion})
	})

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

			// While the server shuts down, termianate all sessions.
			globals.sessionStore.Shutdown()

			// Stop publishing statistics.
			stats.Shutdown()

			// Shutdown the hub. The hub will shutdown topics.
			hubdone := make(chan bool)

			// Wait for the hub to finish.
			<-hubdone
			cancel()

			// Shutdown Extra
			globals.crawler.Shutdown()
			for _, worker := range globals.workers {
				worker.Shutdown()
			}
			globals.scheduler.Shutdown()
			globals.manager.Shutdown()
			for _, ruleset := range globals.cronRuleset {
				ruleset.Shutdown()
			}
			event.Shutdown()
			queue.Shutdown()
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
		logs.Info.Printf("Signal received: '%s', shutting down", sig)
		stop <- true
	}()

	return stop
}

// debugSession is session debug info.
type debugSession struct {
	RemoteAddr string   `json:"remote_addr,omitempty"`
	Ua         string   `json:"ua,omitempty"`
	Uid        string   `json:"uid,omitempty"`
	Sid        string   `json:"sid,omitempty"`
	Clnode     string   `json:"clnode,omitempty"`
	Subs       []string `json:"subs,omitempty"`
}

// debugTopic is a topic debug info.
type debugTopic struct {
	Topic    string   `json:"topic,omitempty"`
	Xorig    string   `json:"xorig,omitempty"`
	IsProxy  bool     `json:"is_proxy,omitempty"`
	PerUser  []string `json:"per_user,omitempty"`
	PerSubs  []string `json:"per_subs,omitempty"`
	Sessions []string `json:"sessions,omitempty"`
}

// debugCachedUser is a user cache entry debug info.
type debugCachedUser struct {
	Uid    string `json:"uid,omitempty"`
	Unread int    `json:"unread,omitempty"`
	Topics int    `json:"topics,omitempty"`
}

// debugDump is server internal state dump for debugging.
type debugDump struct {
	Version   string            `json:"server_version,omitempty"`
	Build     string            `json:"build_id,omitempty"`
	Timestamp time.Time         `json:"ts,omitempty"`
	Sessions  []debugSession    `json:"sessions,omitempty"`
	Topics    []debugTopic      `json:"topics,omitempty"`
	UserCache []debugCachedUser `json:"user_cache,omitempty"`
}

func serveStatus(wrt http.ResponseWriter, _ *http.Request) {
	wrt.Header().Set("Content-Type", "application/json")

	result := &debugDump{
		Version:   version.CurrentVersion,
		Build:     version.Buildstamp,
		Timestamp: types.TimeNow(),
		Sessions:  make([]debugSession, 0, len(globals.sessionStore.sessCache)),
		Topics:    make([]debugTopic, 0, 10),
		UserCache: make([]debugCachedUser, 0, 10),
	}
	// Sessions.
	globals.sessionStore.Range(func(sid string, s *Session) bool {
		keys := make([]string, 0, len(s.subs))
		for tn := range s.subs {
			keys = append(keys, tn)
		}
		sort.Strings(keys)
		var clnode string
		result.Sessions = append(result.Sessions, debugSession{
			RemoteAddr: s.remoteAddr,
			Ua:         s.userAgent,
			Uid:        s.uid.String(),
			Sid:        sid,
			Clnode:     clnode,
			Subs:       keys,
		})
		return true
	})

	_ = json.NewEncoder(wrt).Encode(result)
}
