package server

import (
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/search"
	"go.uber.org/fx"
)

var Modules = fx.Options(
	// controller.Modules,
	// repository.Modules,
	fx.Provide(
		config.NewConfig,
		cache.NewCache,
		rdb.NewClient,
		search.NewClient,
		newController,
		newHTTPServer,
	),
	fx.Invoke(
		bindRoutes,
		RunServer,
	),
)
