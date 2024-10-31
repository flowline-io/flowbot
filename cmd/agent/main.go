package main

import (
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/agent"
	"github.com/flowline-io/flowbot/cmd/agent/ruleset/instruct"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
)

func main() {
	flog.Init()
	flog.Info("[agent] version %s %s", version.Buildtags, version.Buildstamp)

	// check singleton
	utils.CheckSingleton()

	// embed server
	utils.EmbedServer()

	// cron
	instruct.Cron()
	agent.Cron()

	select {}
}
