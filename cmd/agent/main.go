package main

import (
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/agent"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/instruct"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
)

func main() {
	// log
	flog.Init(true)
	flog.Info("[agent] version %s %s", version.Buildtags, version.Buildstamp)

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

	stopSignal := utils.SignalHandler()
	<-stopSignal
}
