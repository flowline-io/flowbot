package apps

import (
	"go.uber.org/fx"
)

// Modules wires App-related providers.
var Modules = fx.Options(
	fx.Provide(
		NewManager,
		NewAPI,
	),
)
