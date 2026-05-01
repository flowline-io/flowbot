package modules

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"go.uber.org/fx"
)

// Modules registers the current module set.
//
// It delegates to internal/bots during the staged bot-to-module migration.
var Modules = fx.Options(bots.Modules)
