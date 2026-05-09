package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/media/fs"
	"github.com/flowline-io/flowbot/pkg/media/minio"
)

var MediaModules = fx.Options(
	fx.Invoke(
		fs.Register,
		minio.Register,
	),
)
