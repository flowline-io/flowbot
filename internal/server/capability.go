package server

import (
	"github.com/flowline-io/flowbot/pkg/ability/archive"
	archiveboxadapter "github.com/flowline-io/flowbot/pkg/ability/archive/archivebox"
	abookmark "github.com/flowline-io/flowbot/pkg/ability/bookmark"
	bookmarkkarakeep "github.com/flowline-io/flowbot/pkg/ability/bookmark/karakeep"
	afinance "github.com/flowline-io/flowbot/pkg/ability/finance"
	financefireflyiii "github.com/flowline-io/flowbot/pkg/ability/finance/fireflyiii"
	akanban "github.com/flowline-io/flowbot/pkg/ability/kanban"
	kanbankanboard "github.com/flowline-io/flowbot/pkg/ability/kanban/kanboard"
	areader "github.com/flowline-io/flowbot/pkg/ability/reader"
	readerminiflux "github.com/flowline-io/flowbot/pkg/ability/reader/miniflux"
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
	if err := afinance.RegisterService("fireflyiii", "fireflyiii", financefireflyiii.New()); err != nil {
		return err
	}
	return archive.RegisterService("archivebox", "archivebox", archiveboxadapter.New())
}
