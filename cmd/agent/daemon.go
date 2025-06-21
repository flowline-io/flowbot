package main

import (
	"context"
	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/collect"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/instruct"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
	"go.uber.org/fx"
)

type Daemon struct {
	hostid   string
	hostname string
}

func NewDaemon(_ config.Type) *Daemon {
	return &Daemon{}
}

func RunDaemon(lc fx.Lifecycle, app *Daemon) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// log
			flog.EnableAlarm = false
			flog.Init(true)
			flog.Info("version %s %s", version.Buildtags, version.Buildstamp)

			// check singleton
			utils.CheckSingleton()

			// embed server
			utils.EmbedServer()

			// check update
			checkUpdate()

			// info
			app.hostid, app.hostname = hostInfo()
			err := client.Online(app.hostid, app.hostname)
			if err != nil {
				flog.Error(err)
			}

			// cron
			instruct.Cron()
			collect.Cron()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// offline
			err := client.Offline(app.hostid)
			if err != nil {
				flog.Error(err)
			}
			return nil
		},
	})
}
