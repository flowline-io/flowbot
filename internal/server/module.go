package server

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/stats"
	"github.com/flowline-io/flowbot/pkg/utils/sets"
	"github.com/flowline-io/flowbot/version"
)

func handleModules(lc fx.Lifecycle, _ *config.Type, _ store.Adapter, _ *redis.Client) error {
	// Initialize modules
	initializeModules(config.App.Bots, config.App.Vendors)

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			// notify after online
			go notifyAll(fmt.Sprintf("flowbot (%s) online", version.Buildtags))

			return nil
		},
		OnStop: func(_ context.Context) error {
			return nil
		},
	})

	return nil
}

// initialize modules
func initializeModules(modulesConfig any, vendorsConfig any) {
	b, err := sonic.Marshal(modulesConfig)
	if err != nil {
		flog.Fatal("Failed to marshal modules: %v", err)
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

	// init modules
	err = module.Init(b)
	if err != nil {
		flog.Fatal("Failed to initialize bot: %v", err)
	}

	// register modules
	registerModules()

	// bootstrap modules
	err = module.Bootstrap()
	if err != nil {
		flog.Fatal("Failed to bootstrap bot: %v", err)
	}

	stats.ModuleTotalCounter().Set(uint64(len(module.List())))
}

// register modules
func registerModules() {
	// register modules
	registerModuless := sets.NewString()
	for name, handler := range module.List() {
		registerModuless.Insert(name)

		state := schema.BotInactive
		if handler.IsReady() {
			state = schema.BotActive
		}
		bot, _ := store.Database.GetBotByName(context.Background(), name)
		if bot == nil {
			bot = &gen.Bot{
				Name:  name,
				State: int(state),
			}
			if _, err := store.Database.CreateBot(context.Background(), bot); err != nil {
				flog.Error(err)
			}
		} else {
			bot.State = int(state)
			err := store.Database.UpdateBot(context.Background(), bot)
			if err != nil {
				flog.Error(err)
			}
		}
	}

	// inactive bot
	list, err := store.Database.GetBots(context.Background())
	if err != nil {
		flog.Error(err)
	}
	for _, bot := range list {
		if !registerModuless.Has(bot.Name) {
			bot.State = int(schema.BotInactive)
			if err := store.Database.UpdateBot(context.Background(), bot); err != nil {
				flog.Error(err)
			}
		}
	}
}
