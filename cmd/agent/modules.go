package main

import (
	"github.com/flowline-io/flowbot/cmd/agent/config"
	"go.uber.org/fx"
)

var Modules = fx.Options(
	fx.Provide(
		config.NewConfig,
		NewDaemon,
	),
	fx.Invoke(
		RunDaemon,
	),
)
