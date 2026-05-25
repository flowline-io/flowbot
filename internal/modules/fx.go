// Package modules provides fx dependency injection registration for all modules.
package modules

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules/example"
	"github.com/flowline-io/flowbot/internal/modules/hub"
	"github.com/flowline-io/flowbot/internal/modules/notify"
	"github.com/flowline-io/flowbot/internal/modules/server"
	"github.com/flowline-io/flowbot/internal/modules/workflow"
)

// Modules registers all interaction modules.
var Modules = fx.Options(
	fx.Invoke(
		example.Register,
		hub.Register,
		notify.Register,
		server.Register,
		workflow.Register,
	),
)
