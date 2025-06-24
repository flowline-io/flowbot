//go:build freebsd || darwin || linux

package shell

import (
	"fmt"
	"strconv"
	"syscall"

	"github.com/flowline-io/flowbot/pkg/flog"
)

func SetUID(uid string) {
	if uid != DefaultUid {
		uidi, err := strconv.Atoi(uid)
		if err != nil {
			flog.Error(fmt.Errorf("invalid uid: %s error: %s", uid, err))
		}
		if err := syscall.Setuid(uidi); err != nil {
			flog.Error(fmt.Errorf("error setting uid: %s error: %s", uid, err))
		}
	}
}

func SetGID(gid string) {
	if gid != DefaultGid {
		gidi, err := strconv.Atoi(gid)
		if err != nil {
			flog.Error(fmt.Errorf("invalid gid: %s error: %s", gid, err))
		}
		if err := syscall.Setgid(gidi); err != nil {
			flog.Error(fmt.Errorf("error setting gid: %s error: %s", gid, err))
		}
	}
}
