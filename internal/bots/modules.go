package bots

import (
	"github.com/flowline-io/flowbot/internal/bots/agent"
	"github.com/flowline-io/flowbot/internal/bots/anki"
	"github.com/flowline-io/flowbot/internal/bots/bookmark"
	"github.com/flowline-io/flowbot/internal/bots/clipboard"
	"github.com/flowline-io/flowbot/internal/bots/cloudflare"
	"github.com/flowline-io/flowbot/internal/bots/dev"
	"github.com/flowline-io/flowbot/internal/bots/finance"
	"github.com/flowline-io/flowbot/internal/bots/gitea"
	"github.com/flowline-io/flowbot/internal/bots/github"
	"github.com/flowline-io/flowbot/internal/bots/kanban"
	"github.com/flowline-io/flowbot/internal/bots/notify"
	"github.com/flowline-io/flowbot/internal/bots/obsidian"
	"github.com/flowline-io/flowbot/internal/bots/okr"
	"github.com/flowline-io/flowbot/internal/bots/reader"
	"github.com/flowline-io/flowbot/internal/bots/search"
	"github.com/flowline-io/flowbot/internal/bots/server"
	"github.com/flowline-io/flowbot/internal/bots/torrent"
	"github.com/flowline-io/flowbot/internal/bots/user"
	"github.com/flowline-io/flowbot/internal/bots/webhook"
	"github.com/flowline-io/flowbot/internal/bots/workflow"
	"go.uber.org/fx"
)

var Modules = fx.Options(
	fx.Invoke(
		agent.Register,
		anki.Register,
		bookmark.Register,
		clipboard.Register,
		cloudflare.Register,
		dev.Register,
		finance.Register,
		gitea.Register,
		github.Register,
		kanban.Register,
		notify.Register,
		obsidian.Register,
		okr.Register,
		reader.Register,
		search.Register,
		server.Register,
		torrent.Register,
		user.Register,
		webhook.Register,
		workflow.Register,
	),
)
