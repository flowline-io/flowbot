package server

import "go.uber.org/fx"

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
		NewHTTPServer,
	),
	fx.Invoke(
		RunServer,
	),
)
