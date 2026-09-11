//go:build !freebsd && !darwin && !linux

// Package shell runs subprocesses with optional uid/gid on supported platforms.
package shell

import (
	"errors"
	"github.com/flowline-io/flowbot/pkg/flog"
)

func SetUID(uid string) {
	if uid != DefaultUid {
		flog.Error(errors.New("setting uid is only supported on unix/linux systems"))
	}
}

func SetGID(gid string) {
	if gid != DefaultGid {
		flog.Error(errors.New("setting gid is only supported on unix/linux systems"))
	}
}
