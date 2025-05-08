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
		// config.NewConfig,
		// zlog.NewZlog,
		// auth.NewEnforcer,
		// task.NewServer,
		// task.NewClient,
		// eventbus.NewManager,
		// NewTaskMux,
		// NewCronScheduler,
		// NewHTTPServer,
		// NewEventSubscriber,
		config.NewConfig,
		rdb.NewClient,
		NewHTTPServer,
	),
	fx.Invoke(
		RunServer,
	),
)
