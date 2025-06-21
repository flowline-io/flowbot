package main

import (
	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/cmd/agent/script"
	"go.uber.org/fx"
)

var Modules = fx.Options(
	fx.Provide(
		config.NewConfig,
		script.NewEngine,
		NewDaemon,
	),
	fx.Invoke(
		RunDaemon,
	),
)
