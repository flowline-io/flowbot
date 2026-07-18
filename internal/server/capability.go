// Package server provides the HTTP API server (Fiber v3) and route handlers.
package server

import (
	"errors"

	"github.com/flowline-io/flowbot/pkg/capability/fireflyiii"
	"github.com/flowline-io/flowbot/pkg/capability/gitea"
	"github.com/flowline-io/flowbot/pkg/capability/github"
	"github.com/flowline-io/flowbot/pkg/capability/kanboard"
	"github.com/flowline-io/flowbot/pkg/capability/karakeep"
	"github.com/flowline-io/flowbot/pkg/capability/memos"
	"github.com/flowline-io/flowbot/pkg/capability/miniflux"
	"github.com/flowline-io/flowbot/pkg/capability/transmission"
	"github.com/flowline-io/flowbot/pkg/capability/trilium"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func initCapabilityHub() error {
	err := errors.Join(
		karakeep.Register("karakeep", karakeep.New()),
		miniflux.Register("miniflux", miniflux.New()),
		kanboard.Register("kanboard", kanboard.New()),
		trilium.Register("trilium", trilium.New()),
		memos.Register("memos", memos.New()),
		fireflyiii.Register("fireflyiii", fireflyiii.New()),
		transmission.Register("transmission", transmission.New()),
		gitea.Register("gitea", gitea.New()),
		github.Register("github", github.New()),
	)
	if err != nil {
		return err
	}
	hub.LogDiscovered()
	return nil
}
