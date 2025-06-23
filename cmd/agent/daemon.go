package main

import (
	"context"
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

	// fx lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// check update
			// checkUpdate()

			// info
			hostid, hostname = hostInfo()

			// heartbeat
			go func() {
				for range heartbeatTicker.C {
					err := client.Online(hostid, hostname)
					if err != nil {
						flog.Error(err)
					}
				}
			}()

			// cron
			instruct.Cron()
			collect.Cron()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// stop heartbeat
			heartbeatTicker.Stop()
			// offline
			err := client.Offline(hostid, hostname)
			if err != nil {
				flog.Error(err)
			}
			return nil
		},
	})
}
