package main

import (
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/agent"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/instruct"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
)

func main() {
	// log
	flog.Init(true)
	flog.Info("version %s %s", version.Buildtags, version.Buildstamp)

	// config
	loadConfig()

	// info
	hostinfo()

	// check singleton
	utils.CheckSingleton()

	// embed server
	utils.EmbedServer()

	// check update
	checkUpdate()

	// cron
	instruct.Cron()
	agent.Cron()

	// notify
	notify.Desktop{}.Notify("flowbot-agent", "started")

	stopSignal := utils.SignalHandler()
	<-stopSignal
}
