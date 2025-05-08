package server

import (
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
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
		cache.NewCache,
		NewHTTPServer,
	),
	fx.Invoke(
		RunServer,
	),
)
