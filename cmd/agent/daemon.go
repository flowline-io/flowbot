package main

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/collect"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/instruct"
	"github.com/flowline-io/flowbot/cmd/agent/script"
	"github.com/flowline-io/flowbot/cmd/agent/startup"
	"github.com/flowline-io/flowbot/pkg/flog"
	"go.uber.org/fx"
)

func RunDaemon(lc fx.Lifecycle, _ *startup.Startup, _ *script.Engine) {
	var hostid, hostname string
	// heartbeat ticker
	heartbeatTicker := time.NewTicker(time.Minute)
	// updater ticker
	updaterTicker := time.NewTicker(time.Minute)

	// fx lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// info
			hostid, hostname = hostInfo()

			// check update
			go func() {
				for range updaterTicker.C {
					checkUpdate()
				}
			}()

			// heartbeat
			go func() {
				for range heartbeatTicker.C {
					err := client.Online(hostid, hostname)
					if err != nil {
						flog.Error(fmt.Errorf("[heartbeat] failed to online, %w", err))
					}
				}
			}()

			// cron
			instruct.Cron()
			collect.Cron()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// stop ticker
			heartbeatTicker.Stop()
			updaterTicker.Stop()
			// offline
			err := client.Offline(hostid, hostname)
			if err != nil {
				flog.Error(fmt.Errorf("failed to offline, %w", err))
			}
			return nil
		},
	})
}
