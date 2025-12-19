package connections

import (
	"go.uber.org/fx"
)

// Modules wires Connection-related providers.
var Modules = fx.Options(
	fx.Provide(
		NewAPI,
	),
)
