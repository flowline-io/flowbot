package server

import (
	"github.com/flowline-io/flowbot/internal/agents/mcp"
	"github.com/flowline-io/flowbot/internal/bots"
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
	fx.Provide(
		config.NewConfig,
		cache.NewCache,
		rdb.NewClient,
		search.NewClient,
		event.NewRouter,
		event.NewSubscriber,
		event.NewPublisher,
		slack.NewDriver,
		mcp.NewServer,
		mcp.NewSSEServer,
		newController,
		newDatabaseAdapter,
		newHTTPServer,
	),
	fx.Invoke(
		handleRoutes,
		handleEvents,
		handleChatbot,
		handlePlatform,
		RunServer,
	),
)
