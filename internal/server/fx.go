package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules"
	"github.com/flowline-io/flowbot/internal/modules/bookmark"
	"github.com/flowline-io/flowbot/internal/modules/gitea"
	"github.com/flowline-io/flowbot/internal/modules/kanban"
	"github.com/flowline-io/flowbot/internal/modules/reader"
	serverModule "github.com/flowline-io/flowbot/internal/modules/server"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/profiling"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/search"
	"github.com/flowline-io/flowbot/pkg/trace"
)

var Modules = fx.Options(
	metrics.Module(),
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
		newAuditor,
	),
	fx.Invoke(
		setServerCacheStore,
		setModuleServerCacheStore,
		setModuleCacheStore,
		setBookmarkCacheStore,
		setReaderCacheStore,
		setKanbanCacheStore,
		setGiteaCacheStore,
		setRouteAuditor,
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

func setBookmarkCacheStore(store *cache.RedisStore) {
	bookmark.SetCacheStore(store)
}

func setReaderCacheStore(store *cache.RedisStore) {
	reader.SetCacheStore(store)
}

func setKanbanCacheStore(store *cache.RedisStore) {
	kanban.SetCacheStore(store)
}

func setGiteaCacheStore(store *cache.RedisStore) {
	gitea.SetCacheStore(store)
}

// newAuditor creates an audit.Auditor from the global store database.
// Returns nil if the database is not yet initialized.
func newAuditor() audit.Auditor {
	if store.Database == nil || store.Database.GetDB() == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewAuditStore(client)
}

// setRouteAuditor injects the global auditor into the route package
// for auth failure audit logging in the Authorize middleware.
func setRouteAuditor(a audit.Auditor) {
	route.SetAuditor(a)
}
