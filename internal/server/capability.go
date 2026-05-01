package server

import (
	"github.com/flowline-io/flowbot/pkg/ability/archive"
	archiveboxadapter "github.com/flowline-io/flowbot/pkg/ability/archive/archivebox"
	abookmark "github.com/flowline-io/flowbot/pkg/ability/bookmark"
	bookmarkkarakeep "github.com/flowline-io/flowbot/pkg/ability/bookmark/karakeep"
)

func initCapabilityHub() error {
	if err := abookmark.RegisterService("karakeep", "karakeep", bookmarkkarakeep.New()); err != nil {
		return err
	}
	return archive.RegisterService("archivebox", "archivebox", archiveboxadapter.New())
}
