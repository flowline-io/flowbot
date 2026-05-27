package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/executor/runtime/shell"
)

// ReexecModules registers all reexec handlers via fx.
var ReexecModules = fx.Options(
	fx.Invoke(
		shell.Register,
	),
)
