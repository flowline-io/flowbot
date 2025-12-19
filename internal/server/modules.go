package server

import (
	"github.com/flowline-io/flowbot/internal/apps"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/connections"
	"github.com/flowline-io/flowbot/internal/flows"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
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
	flows.Modules,
	apps.Modules,
	connections.Modules,
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
