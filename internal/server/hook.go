package server

import (
	"fmt"
	"github.com/flowline-io/flowbot/version"
	"strings"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/channels"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/stats"
	jsoniter "github.com/json-iterator/go"
)

// hook

func hookBot(botsConfig interface{}, vendorsConfig interface{}) {
	b, err := jsoniter.Marshal(botsConfig)
	if err != nil {
		flog.Fatal("Failed to marshal bots: %v", err)
	}
	v, err := jsoniter.Marshal(vendorsConfig)
	if err != nil {
		flog.Fatal("Failed to marshal vendors: %v", err)
	}

	// set vendors configs
	providers.Configs = v

	// init bots
	err = bots.Init(b)
	if err != nil {
		flog.Fatal("Failed to initialize bot: %v", err)
	}

	// register bots
	initializeBot()

	// bootstrap bots
	err = bots.Bootstrap()
	if err != nil {
		flog.Fatal("Failed to bootstrap bot: %v", err)
	}

	// bot cron
	globals.cronRuleset, err = bots.Cron()
	if err != nil {
		flog.Fatal("Failed to bot cron: %v", err)
	}

	// bot workflow
	err = initializeWorkflow()
	if err != nil {
		flog.Fatal("Failed to initialize workflow: %v", err)
	}

	// stats register
	stats.RegisterInt(types.BotTotalStatsName)
	stats.RegisterInt(types.BotRunInputTotalStatsName)
	stats.RegisterInt(types.BotRunGroupTotalStatsName)
	stats.RegisterInt(types.BotRunAgentTotalStatsName)
	stats.RegisterInt(types.BotRunCommandTotalStatsName)
	stats.RegisterInt(types.BotRunConditionTotalStatsName)
	stats.RegisterInt(types.BotRunCronTotalStatsName)
	stats.RegisterInt(types.BotRunFormTotalStatsName)
	stats.RegisterInt(types.BotTriggerPipelineTotalStatsName)

	stats.Set(types.BotTotalStatsName, int64(len(bots.List())))
}

func hookChannel() {
	err := channels.Init()
	if err != nil {
		flog.Fatal("Failed to initialize channel: %v", err)
	}

	err = initializeChannels()
	if err != nil {
		flog.Fatal("Failed to create or update channels: %v", err)
	}

	err = initializeCrawler()
	if err != nil {
		flog.Fatal("Failed to initialize crawler: %v", err)
	}

	// stats register
	stats.RegisterInt("ChannelTotal")
	stats.RegisterInt("ChannelPublishTotal")

	stats.Set("ChannelTotal", int64(len(channels.List())))
}

func hookIncomingMessage(caller *platforms.Caller, msg protocol.Event) {
	// update online status
	onlineStatus(msg)
	// check grp or p2p
	if strings.HasSuffix(msg.DetailType, ".direct") {
		directIncomingMessage(caller, msg)
	}
	if strings.HasSuffix(msg.DetailType, ".group") {
		groupIncomingMessage(caller, msg)
	}
}

func hookStarted() {
	// notify after online
	go notifyAll(fmt.Sprintf("flowbot (%s) online", version.Buildtags))
}
