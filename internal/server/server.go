package server

import (
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"

	// bots
	_ "github.com/flowline-io/flowbot/internal/bots/anki"
	_ "github.com/flowline-io/flowbot/internal/bots/clipboard"
	_ "github.com/flowline-io/flowbot/internal/bots/cloudflare"
	_ "github.com/flowline-io/flowbot/internal/bots/dev"
	_ "github.com/flowline-io/flowbot/internal/bots/finance"
	_ "github.com/flowline-io/flowbot/internal/bots/flowkit"
	_ "github.com/flowline-io/flowbot/internal/bots/gitea"
	_ "github.com/flowline-io/flowbot/internal/bots/github"
	_ "github.com/flowline-io/flowbot/internal/bots/leetcode"
	_ "github.com/flowline-io/flowbot/internal/bots/markdown"
	_ "github.com/flowline-io/flowbot/internal/bots/obsidian"
	_ "github.com/flowline-io/flowbot/internal/bots/okr"
	_ "github.com/flowline-io/flowbot/internal/bots/pocket"
	_ "github.com/flowline-io/flowbot/internal/bots/server"
	_ "github.com/flowline-io/flowbot/internal/bots/share"
	_ "github.com/flowline-io/flowbot/internal/bots/subscribe"
	_ "github.com/flowline-io/flowbot/internal/bots/torrent"
	_ "github.com/flowline-io/flowbot/internal/bots/user"
	_ "github.com/flowline-io/flowbot/internal/bots/webhook"
	_ "github.com/flowline-io/flowbot/internal/bots/workflow"

	// File upload handlers
	_ "github.com/flowline-io/flowbot/pkg/media/fs"
	_ "github.com/flowline-io/flowbot/pkg/media/minio"

	// Notify
	_ "github.com/flowline-io/flowbot/pkg/notify/ntfy"
	_ "github.com/flowline-io/flowbot/pkg/notify/pushover"
)

const (
	// Base URL path for serving the streaming API.
	defaultApiPath = "/"
)

func Run() {
	// initialize
	if err := initialize(); err != nil {
		flog.Fatal("initialize %v", err)
	}
	// serve
	if err := listenAndServe(httpApp, config.App.Listen, tlsConfig, stopSignal); err != nil {
		flog.Fatal("listenAndServe %v", err)
	}
}
