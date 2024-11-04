package main

import (
	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/collect"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/instruct"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
)

func main() {
	// log
	flog.Init(true)
	flog.Info("version %s %s", version.Buildtags, version.Buildstamp)

	// config
	loadConfig()

	// check singleton
	utils.CheckSingleton()

	// embed server
	utils.EmbedServer()

	// check update
	checkUpdate()

	// info
	hostid, hostname := hostinfo()
	err := client.Online(hostid, hostname)
	if err != nil {
		flog.Error(err)
	}

	// cron
	instruct.Cron()
	collect.Cron()

	stopSignal := utils.SignalHandler()
	<-stopSignal

	// offline
	err = client.Offline(hostid)
	if err != nil {
		flog.Error(err)
	}
}
