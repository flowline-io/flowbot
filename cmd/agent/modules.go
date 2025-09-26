package main

import (
	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/cmd/agent/script"
	"github.com/flowline-io/flowbot/cmd/agent/startup"
	"go.uber.org/fx"
)

var Modules = fx.Options(
	fx.Provide(
		config.NewConfig,
		script.NewEngine,
		startup.NewStartup,
	),
	fx.Invoke(
		RunDaemon,
		tickMetrics,
	),
)
