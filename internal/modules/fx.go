// Package modules provides fx dependency injection registration for all modules.
package modules

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules/bookmark"
	"github.com/flowline-io/flowbot/internal/modules/example"
	"github.com/flowline-io/flowbot/internal/modules/gitea"
	"github.com/flowline-io/flowbot/internal/modules/github"
	"github.com/flowline-io/flowbot/internal/modules/hub"
	"github.com/flowline-io/flowbot/internal/modules/kanban"
	"github.com/flowline-io/flowbot/internal/modules/notify"
	"github.com/flowline-io/flowbot/internal/modules/reader"
	"github.com/flowline-io/flowbot/internal/modules/resourcechain"
	"github.com/flowline-io/flowbot/internal/modules/server"
	"github.com/flowline-io/flowbot/internal/modules/workflow"
)

// Modules registers all interaction modules.
var Modules = fx.Options(
	fx.Invoke(
		bookmark.Register,
		example.Register,
		gitea.Register,
		github.Register,
		hub.Register,
		kanban.Register,
		notify.Register,
		reader.Register,
		resourcechain.Register,
		server.Register,
		workflow.Register,
	),
)
