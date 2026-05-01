package server

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
	"github.com/flowline-io/flowbot/version"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func handleModules(lc fx.Lifecycle, _ config.Type, _ store.Adapter, _ *redis.Client) error {
	// Initialize bots
	initializeModules(config.App.Bots, config.App.Vendors)

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
func initializeModules(modulesConfig any, vendorsConfig any) {
	b, err := sonic.Marshal(modulesConfig)
	if err != nil {
		flog.Fatal("Failed to marshal bots: %v", err)
	}
	v, err := sonic.Marshal(vendorsConfig)
	if err != nil {
		flog.Fatal("Failed to marshal vendors: %v", err)
	}

	// set vendors configs
	providers.Configs = v

	// init homelab app registry
	if err := initHomelabRegistry(config.App.Homelab); err != nil {
		flog.Fatal("Failed to initialize homelab registry: %v", err)
	}

	// init capability hub
	if err := initCapabilityHub(); err != nil {
		flog.Fatal("Failed to initialize capability hub: %v", err)
	}

	// init bots
	err = module.Init(b)
	if err != nil {
		flog.Fatal("Failed to initialize bot: %v", err)
	}

	// register bots
	registerModules()

	// bootstrap bots
	err = module.Bootstrap()
	if err != nil {
		flog.Fatal("Failed to bootstrap bot: %v", err)
	}

	// bot cron
	globals.cronRuleset, err = module.Cron()
	if err != nil {
		flog.Fatal("Failed to bot cron: %v", err)
	}

	stats.ModuleTotalCounter().Set(uint64(len(module.List())))
	rdb.SetMetricsInt64(stats.ModuleTotalStatsName, int64(len(module.List())))
}

// register bots
func registerModules() {
	// register bots
	registerModuless := sets.NewString()
	for name, handler := range module.List() {
		registerModuless.Insert(name)

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
		if !registerModuless.Has(bot.Name) {
			bot.State = model.BotInactive
			if err := store.Database.UpdateBot(bot); err != nil {
				flog.Error(err)
			}
		}
	}
}
