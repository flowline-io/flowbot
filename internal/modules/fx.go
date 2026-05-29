// Package modules provides fx dependency injection registration for all modules.
package modules

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules/example"
	"github.com/flowline-io/flowbot/internal/modules/hub"
)

// Modules registers all interaction modules.
var Modules = fx.Options(
	fx.Invoke(
		example.Register,
		hub.Register,
	),
)
