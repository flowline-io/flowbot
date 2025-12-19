package server

import (
	"github.com/flowline-io/flowbot/internal/apps"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/connections"
	"github.com/flowline-io/flowbot/internal/flows"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/search"
	"go.uber.org/fx"
)

var Modules = fx.Options(
	bots.Modules,
	NotifyModules,
	MediaModules,
	fx.Provide(
		config.NewConfig,
		cache.NewCache,
		rdb.NewClient,
		search.NewClient,
		event.NewRouter,
		event.NewSubscriber,
		event.NewPublisher,
		slack.NewDriver,
		newController,
		newDatabaseAdapter,
		newHTTPServer,
		// Flow modules
		func(storeAdapter store.Adapter) *flows.Engine {
			return flows.NewEngine(storeAdapter)
		},
		func(storeAdapter store.Adapter) *flows.RateLimiter {
			return flows.NewRateLimiter(storeAdapter)
		},
		func(engine *flows.Engine, storeAdapter store.Adapter) (*flows.QueueManager, error) {
			return flows.NewQueueManager(storeAdapter, engine)
		},
		func(storeAdapter store.Adapter, queue *flows.QueueManager) *flows.Poller {
			return flows.NewPoller(storeAdapter, queue)
		},
		func(engine *flows.Engine, rateLimiter *flows.RateLimiter, storeAdapter store.Adapter, queue *flows.QueueManager) *flows.API {
			return flows.NewAPI(engine, rateLimiter, storeAdapter, queue)
		},
		// App modules
		func(storeAdapter store.Adapter) (*apps.Manager, error) {
			return apps.NewManager(storeAdapter)
		},
		func(manager *apps.Manager, storeAdapter store.Adapter) *apps.API {
			return apps.NewAPI(manager, storeAdapter)
		},
		// Connection modules
		func(storeAdapter store.Adapter) *connections.API {
			return connections.NewAPI(storeAdapter)
		},
	),
	fx.Invoke(
		handleRoutes,
		handleEvents,
		handleChatbot,
		handlePlatform,
		handleFlowQueue,
		handleFlowPoller,
		RunServer,
	),
)
