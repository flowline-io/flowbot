package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules"
	serverModule "github.com/flowline-io/flowbot/internal/modules/server"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/profiling"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/search"
	"github.com/flowline-io/flowbot/pkg/trace"
)

var Modules = fx.Options(
	modules.Modules,
	NotifyModules,
	MediaModules,
	fx.Provide(
		config.NewConfig,
		cache.NewCache,
		rdb.NewClient,
		cache.NewRedisStore,
		search.NewClient,
		event.NewRouter,
		event.NewSubscriber,
		event.NewPublisher,
		slack.NewDriver,
		trace.NewTracerProvider,
		newController,
		newDatabaseAdapter,
		newHTTPServer,
	),
	fx.Invoke(
		setServerCacheStore,
		setModuleServerCacheStore,
		setModuleCacheStore,
		handleRoutes,
		handleEvents,
		handleModules,
		handlePlatform,
		initPipeline,
		RunServer,
		profiling.NewProfiler,
	),
)

func setServerCacheStore(store *cache.RedisStore) {
	SetCacheStore(store)
}

func setModuleServerCacheStore(store *cache.RedisStore) {
	serverModule.SetCacheStore(store)
}

func setModuleCacheStore(store *cache.RedisStore) {
	module.SetCacheStore(store)
}
