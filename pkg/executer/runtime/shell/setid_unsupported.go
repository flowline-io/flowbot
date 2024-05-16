//go:build !freebsd && !darwin && !linux

package shell

import (
	"github.com/rs/zerolog/log"
)

func SetUID(uid string) {
	if uid != DefaultUid {
		log.Fatal().Msgf("setting uid is only supported on unix/linux systems")
	}
}

func SetGID(gid string) {
	if gid != DefaultGid {
		log.Fatal().Msgf("setting gid is only supported on unix/linux systems")
	}
}
