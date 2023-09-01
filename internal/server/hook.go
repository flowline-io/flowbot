package server

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/channels"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/stats"
	json "github.com/json-iterator/go"
	"strings"

	// bots
	_ "github.com/flowline-io/flowbot/internal/bots/anki"
	_ "github.com/flowline-io/flowbot/internal/bots/attendance"
	_ "github.com/flowline-io/flowbot/internal/bots/clipboard"
	_ "github.com/flowline-io/flowbot/internal/bots/cloudflare"
	_ "github.com/flowline-io/flowbot/internal/bots/dev"
	_ "github.com/flowline-io/flowbot/internal/bots/download"
	_ "github.com/flowline-io/flowbot/internal/bots/finance"
	_ "github.com/flowline-io/flowbot/internal/bots/genshin"
	_ "github.com/flowline-io/flowbot/internal/bots/github"
	_ "github.com/flowline-io/flowbot/internal/bots/gpt"
	_ "github.com/flowline-io/flowbot/internal/bots/iot"
	_ "github.com/flowline-io/flowbot/internal/bots/leetcode"
	_ "github.com/flowline-io/flowbot/internal/bots/linkit"
	_ "github.com/flowline-io/flowbot/internal/bots/markdown"
	_ "github.com/flowline-io/flowbot/internal/bots/mtg"
	_ "github.com/flowline-io/flowbot/internal/bots/notion"
	_ "github.com/flowline-io/flowbot/internal/bots/obsidian"
	_ "github.com/flowline-io/flowbot/internal/bots/okr"
	_ "github.com/flowline-io/flowbot/internal/bots/pocket"
	_ "github.com/flowline-io/flowbot/internal/bots/qr"
	_ "github.com/flowline-io/flowbot/internal/bots/queue"
	_ "github.com/flowline-io/flowbot/internal/bots/rust"
	_ "github.com/flowline-io/flowbot/internal/bots/server"
	_ "github.com/flowline-io/flowbot/internal/bots/share"
	_ "github.com/flowline-io/flowbot/internal/bots/subscribe"
	_ "github.com/flowline-io/flowbot/internal/bots/url"
	_ "github.com/flowline-io/flowbot/internal/bots/web"
	_ "github.com/flowline-io/flowbot/internal/bots/webhook"
	_ "github.com/flowline-io/flowbot/internal/bots/workflow"

	// cache
	_ "github.com/flowline-io/flowbot/pkg/cache"
)

// hook

func hookBot(botsConfig interface{}, vendorsConfig interface{}) {
	b, err := json.Marshal(botsConfig)
	if err != nil {
		flog.Fatal("Failed to marshal bots: %v", err)
	}
	v, err := json.Marshal(vendorsConfig)
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

	// bootstrap bots
	err = bots.Bootstrap()
	if err != nil {
		flog.Fatal("Failed to bootstrap bot: %v", err)
	}

	// bot father
	err = initializeBotFather()
	if err != nil {
		flog.Fatal("Failed to create or update bot father: %v", err)
	}

	// bot users
	err = initializeBotUsers()
	if err != nil {
		flog.Fatal("Failed to create or update bot users: %v", err)
	}

	// bot cron
	globals.cronRuleset, err = bots.Cron(botSend)
	if err != nil {
		flog.Fatal("Failed to bot cron: %v", err)
	}

	// bot workflow
	err = initializeWorkflow()
	if err != nil {
		flog.Fatal("Failed to initialize workflow: %v", err)
	}

	// stats register
	stats.RegisterInt("BotTotal")
	stats.RegisterInt("BotRunInputTotal")
	stats.RegisterInt("BotRunGroupTotal")
	stats.RegisterInt("BotRunAgentTotal")
	stats.RegisterInt("BotRunCommandTotal")
	stats.RegisterInt("BotRunConditionTotal")
	stats.RegisterInt("BotRunCronTotal")
	stats.RegisterInt("BotRunFormTotal")
	stats.RegisterInt("BotTriggerPipelineTotal")

	stats.Set("BotTotal", int64(len(bots.List())))
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

func hookHandleIncomingMessage(t *Topic, msg *ClientComMessage) {
	// update online status
	onlineStatus(msg.AsUser)
	// check grp or p2p
	if strings.HasPrefix(msg.Original, "grp") {
		groupIncomingMessage(t, msg, types.GroupEventReceive)
	} else {
		botIncomingMessage(t, msg)
	}
}

func hookHandleGroupEvent(t *Topic, msg *ClientComMessage, event int) {
	if strings.HasPrefix(msg.Original, "grp") {
		switch types.GroupEvent(event) {
		case types.GroupEventJoin:
			msg.AsUser = msg.Set.MsgSetQuery.Sub.User
		case types.GroupEventExit:
			msg.AsUser = msg.Del.User
		}
		//user, err := tstore.Users.Get(types.ParseUserId(msg.AsUser))
		//if err != nil {
		//	flog.Error(err)
		//}
		// Current user is bot
		//if isBotUser(user) {
		//	return
		//}
		groupIncomingMessage(t, msg, types.GroupEvent(event))
	}
}

func hookMounted() {
	// notify after reboot
	go notifyAfterReboot()
}

func hookEvent() {
	onSendEvent()
	onPushInstruct()
}
