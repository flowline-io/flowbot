package server

import (
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/search"
	"go.uber.org/fx"
)

var Modules = fx.Options(
	BotsModules,
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
		workflow.NewQueue,
		workflow.NewManager,
		workflow.NewCronTaskManager,
		newController,
		newDatabaseAdapter,
		newHTTPServer,
	),
	fx.Invoke(
		bindRoutes,
		handleEvents,
		handleChatbot,
		RunServer,
	),
)
