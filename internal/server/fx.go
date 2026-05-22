package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/metrics"
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

// newAuditor creates an audit.Auditor from the global store database.
// Returns nil if the database is not yet initialized.
func newAuditor() audit.Auditor {
	if storepkg.Database == nil || storepkg.Database.GetDB() == nil {
		return nil
	}
	client, ok := storepkg.Database.GetDB().(*storepkg.Client)
	if !ok {
		return nil
	}
	return storepkg.NewAuditStore(client)
}

// setRouteAuditor injects the global auditor into the route package
// for auth failure audit logging in the Authorize middleware.
func setRouteAuditor(a audit.Auditor) {
	route.SetAuditor(a)
}
