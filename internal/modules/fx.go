package modules

import (
	"github.com/flowline-io/flowbot/internal/modules/bookmark"
	"github.com/flowline-io/flowbot/internal/modules/dev"
	"github.com/flowline-io/flowbot/internal/modules/gitea"
	"github.com/flowline-io/flowbot/internal/modules/github"
	"github.com/flowline-io/flowbot/internal/modules/hub"
	"github.com/flowline-io/flowbot/internal/modules/kanban"
	"github.com/flowline-io/flowbot/internal/modules/notify"
	"github.com/flowline-io/flowbot/internal/modules/reader"
	"github.com/flowline-io/flowbot/internal/modules/server"
	"github.com/flowline-io/flowbot/internal/modules/webhook"
	"github.com/flowline-io/flowbot/internal/modules/workflow"
	"go.uber.org/fx"
)

// Modules registers all interaction modules.
var Modules = fx.Options(
	fx.Invoke(
		bookmark.Register,
		dev.Register,
		gitea.Register,
		github.Register,
		hub.Register,
		kanban.Register,
		notify.Register,
		reader.Register,
		server.Register,
		webhook.Register,
		workflow.Register,
	),
)
