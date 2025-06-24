//go:build !freebsd && !darwin && !linux

package shell

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/flog"
)

func SetUID(uid string) {
	if uid != DefaultUid {
		flog.Error(fmt.Errorf("setting uid is only supported on unix/linux systems"))
	}
}

func SetGID(gid string) {
	if gid != DefaultGid {
		flog.Error(fmt.Errorf("setting gid is only supported on unix/linux systems"))
	}
}
