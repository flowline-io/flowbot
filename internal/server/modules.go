package server

import (
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"go.uber.org/fx"
)

var Modules = fx.Options(
	// controller.Modules,
	// repository.Modules,
	fx.Provide(
		config.NewConfig,
		rdb.NewClient,
		newController,
		newHTTPServer,
	),
	fx.Invoke(
		bindRoutes,
		RunServer,
	),
)
