package modules

import (
	"github.com/flowline-io/flowbot/internal/modules/agent"
	"github.com/flowline-io/flowbot/internal/modules/anki"
	"github.com/flowline-io/flowbot/internal/modules/archive"
	"github.com/flowline-io/flowbot/internal/modules/bookmark"
	"github.com/flowline-io/flowbot/internal/modules/clipboard"
	"github.com/flowline-io/flowbot/internal/modules/cloudflare"
	"github.com/flowline-io/flowbot/internal/modules/dev"
	"github.com/flowline-io/flowbot/internal/modules/finance"
	"github.com/flowline-io/flowbot/internal/modules/gitea"
	"github.com/flowline-io/flowbot/internal/modules/github"
	"github.com/flowline-io/flowbot/internal/modules/hub"
	"github.com/flowline-io/flowbot/internal/modules/kanban"
	"github.com/flowline-io/flowbot/internal/modules/notify"
	"github.com/flowline-io/flowbot/internal/modules/reader"
	"github.com/flowline-io/flowbot/internal/modules/search"
	"github.com/flowline-io/flowbot/internal/modules/server"
	"github.com/flowline-io/flowbot/internal/modules/torrent"
	"github.com/flowline-io/flowbot/internal/modules/user"
	"github.com/flowline-io/flowbot/internal/modules/webhook"
	"github.com/flowline-io/flowbot/internal/modules/workflow"
	"go.uber.org/fx"
)

// Modules registers all interaction modules.
var Modules = fx.Options(
	fx.Invoke(
		agent.Register,
		anki.Register,
		archive.Register,
		bookmark.Register,
		clipboard.Register,
		cloudflare.Register,
		dev.Register,
		finance.Register,
		gitea.Register,
		github.Register,
		hub.Register,
		kanban.Register,
		notify.Register,
		reader.Register,
		search.Register,
		server.Register,
		torrent.Register,
		user.Register,
		webhook.Register,
		workflow.Register,
	),
)
