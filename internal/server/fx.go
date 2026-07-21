package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/profiling"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

var Modules = fx.Options(
	flog.FxModule,
	metrics.Module(),
	modules.Modules,
	NotifyModules,
	MediaModules,
	OAuthModules,
	ReexecModules,
	fx.Provide(
		config.NewConfig,
		cache.NewCache,
		rdb.NewClient,
		cache.NewRedisStore,
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
		setAgentMetricsCollector,
		handleRoutes,
		handleEvents,
		initPipeline,
		initWorkflow,
		initAgentAbility,
		initClipAbility,
		initChatAgentScheduler,
		handleModules,
		handlePlatform,
		RunServer,
		profiling.NewProfiler,
		initPageDataCleanup,
	),
)

func setServerCacheStore(store *cache.RedisStore) {
	SetCacheStore(store)
}

// setAgentMetricsCollector installs the process-wide agent harness metrics collector.
func setAgentMetricsCollector(ac *metrics.AgentCollector) {
	metrics.SetDefaultAgentCollector(ac)
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
