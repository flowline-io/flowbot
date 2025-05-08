package server

import (
	"github.com/flowline-io/flowbot/pkg/media/fs"
	"github.com/flowline-io/flowbot/pkg/media/minio"
	"go.uber.org/fx"
)

var MediaModules = fx.Options(
	fx.Invoke(
		fs.Register,
		minio.Register,
	),
)
