package server

import (
	"encoding/json"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	extraMysql "github.com/flowline-io/flowbot/internal/store/mysql"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/channels"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/queue"
	"github.com/flowline-io/flowbot/pkg/route"
	"net/http"
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

func hookMux() *http.ServeMux {
	// Webservice
	wc := route.NewContainer()
	for _, bot := range bots.List() {
		if ws := bot.Webservice(); ws != nil {
			wc.Add(ws)
		}
	}
	route.AddSwagger(wc)
	mux := wc.ServeMux

	mux.Handle("/extra/", newRouter())
	mux.Handle("/app/", newWebappRouter())
	mux.Handle("/u/", newUrlRouter())
	mux.Handle("/d/", newDownloadRouter())

	return mux
}

func hookStore() {
	// init cache
	cache.InitCache()
	// init database
	extraMysql.Init()
	store.Init()
}

func hookBot(jsconfig json.RawMessage, vc json.RawMessage) {
	// set vendors configs
	providers.Configs = vc

	// init bots
	err := bots.Init(jsconfig)
	if err != nil {
		logs.Err.Fatal("Failed to initialize bot:", err)
	}

	// bootstrap bots
	err = bots.Bootstrap()
	if err != nil {
		logs.Err.Fatal("Failed to bootstrap bot:", err)
	}

	// bot father
	err = initializeBotFather()
	if err != nil {
		logs.Err.Fatal("Failed to create or update bot father:", err)
	}

	// bot users
	err = initializeBotUsers()
	if err != nil {
		logs.Err.Fatal("Failed to create or update bot users:", err)
	}

	// bot cron
	globals.cronRuleset, err = bots.Cron(botSend)
	if err != nil {
		logs.Err.Fatal("Failed to bot cron:", err)
	}

	// bot workflow
	err = initializeWorkflow()
	if err != nil {
		logs.Err.Fatal("Failed to initialize workflow:", err)
	}

	// stats register
	statsRegisterInt("BotTotal")
	statsRegisterInt("BotRunInputTotal")
	statsRegisterInt("BotRunGroupTotal")
	statsRegisterInt("BotRunAgentTotal")
	statsRegisterInt("BotRunCommandTotal")
	statsRegisterInt("BotRunConditionTotal")
	statsRegisterInt("BotRunCronTotal")
	statsRegisterInt("BotRunFormTotal")
	statsRegisterInt("BotTriggerPipelineTotal")

	statsSet("BotTotal", int64(len(bots.List())))
}

func hookChannel() {
	err := channels.Init()
	if err != nil {
		logs.Err.Fatal("Failed to initialize channel:", err)
	}

	err = initializeChannels()
	if err != nil {
		logs.Err.Fatal("Failed to create or update channels:", err)
	}

	err = initializeCrawler()
	if err != nil {
		logs.Err.Fatal("Failed to initialize crawler:", err)
	}

	// stats register
	statsRegisterInt("ChannelTotal")
	statsRegisterInt("ChannelPublishTotal")

	statsSet("ChannelTotal", int64(len(channels.List())))
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
		//	logs.Err.Println(err)
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

func hookQueue() {
	queue.InitMessageQueue(NewAsyncMessageConsumer())
}

func hookEvent() {
	onSendEvent()
	onPushInstruct()
}
