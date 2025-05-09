package server

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/bots/agent"
	"github.com/flowline-io/flowbot/internal/bots/anki"
	"github.com/flowline-io/flowbot/internal/bots/bookmark"
	"github.com/flowline-io/flowbot/internal/bots/clipboard"
	"github.com/flowline-io/flowbot/internal/bots/cloudflare"
	"github.com/flowline-io/flowbot/internal/bots/dev"
	"github.com/flowline-io/flowbot/internal/bots/finance"
	"github.com/flowline-io/flowbot/internal/bots/gitea"
	"github.com/flowline-io/flowbot/internal/bots/github"
	"github.com/flowline-io/flowbot/internal/bots/kanban"
	"github.com/flowline-io/flowbot/internal/bots/notify"
	"github.com/flowline-io/flowbot/internal/bots/obsidian"
	"github.com/flowline-io/flowbot/internal/bots/okr"
	"github.com/flowline-io/flowbot/internal/bots/reader"
	"github.com/flowline-io/flowbot/internal/bots/search"
	"github.com/flowline-io/flowbot/internal/bots/server"
	"github.com/flowline-io/flowbot/internal/bots/torrent"
	"github.com/flowline-io/flowbot/internal/bots/user"
	"github.com/flowline-io/flowbot/internal/bots/webhook"
	"github.com/flowline-io/flowbot/internal/bots/workflow"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
	"github.com/flowline-io/flowbot/version"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

var BotsModules = fx.Options(
	fx.Invoke(
		agent.Register,
		anki.Register,
		bookmark.Register,
		clipboard.Register,
		cloudflare.Register,
		dev.Register,
		finance.Register,
		gitea.Register,
		github.Register,
		kanban.Register,
		notify.Register,
		obsidian.Register,
		okr.Register,
		reader.Register,
		search.Register,
		server.Register,
		torrent.Register,
		user.Register,
		webhook.Register,
		workflow.Register,
	),
)

func handleChatbot(lc fx.Lifecycle, _ config.Type, _ store.Adapter, _ *redis.Client) error {
	// Initialize bots
	initializeBot(config.App.Bots, config.App.Vendors)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// notify after online
			go notifyAll(fmt.Sprintf("flowbot (%s) online", version.Buildtags))

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return nil
}

// initialize bots
func initializeBot(botsConfig interface{}, vendorsConfig interface{}) {
	b, err := sonic.Marshal(botsConfig)
	if err != nil {
		flog.Fatal("Failed to marshal bots: %v", err)
	}
	v, err := sonic.Marshal(vendorsConfig)
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
	registerBot()

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

	stats.BotTotalCounter().Set(uint64(len(bots.List())))
	rdb.SetInt64(stats.BotTotalStatsName, int64(len(bots.List())))
}

// register bots
func registerBot() {
	// register bots
	registerBots := sets.NewString()
	for name, handler := range bots.List() {
		registerBots.Insert(name)

		state := model.BotInactive
		if handler.IsReady() {
			state = model.BotActive
		}
		bot, _ := store.Database.GetBotByName(name)
		if bot == nil {
			bot = &model.Bot{
				Name:  name,
				State: state,
			}
			if _, err := store.Database.CreateBot(bot); err != nil {
				flog.Error(err)
			}
		} else {
			bot.State = state
			err := store.Database.UpdateBot(bot)
			if err != nil {
				flog.Error(err)
			}
		}
	}

	// inactive bot
	list, err := store.Database.GetBots()
	if err != nil {
		flog.Error(err)
	}
	for _, bot := range list {
		if !registerBots.Has(bot.Name) {
			bot.State = model.BotInactive
			if err := store.Database.UpdateBot(bot); err != nil {
				flog.Error(err)
			}
		}
	}
}
