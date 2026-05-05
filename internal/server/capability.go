package server

import (
	abookmark "github.com/flowline-io/flowbot/pkg/ability/bookmark"
	bookmarkkarakeep "github.com/flowline-io/flowbot/pkg/ability/bookmark/karakeep"
	akanban "github.com/flowline-io/flowbot/pkg/ability/kanban"
	kanbankanboard "github.com/flowline-io/flowbot/pkg/ability/kanban/kanboard"
	areader "github.com/flowline-io/flowbot/pkg/ability/reader"
	readerminiflux "github.com/flowline-io/flowbot/pkg/ability/reader/miniflux"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func initCapabilityHub() error {
	if err := abookmark.RegisterService("karakeep", "karakeep", bookmarkkarakeep.New()); err != nil {
		return err
	}
	if err := areader.RegisterService("miniflux", "miniflux", readerminiflux.New()); err != nil {
		return err
	}
	if err := akanban.RegisterService("kanboard", "kanboard", kanbankanboard.New()); err != nil {
		return err
	}
	hub.LogDiscovered()
	return nil
}
