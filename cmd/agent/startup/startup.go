package startup

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
	"go.uber.org/fx"
)

type Startup struct{}

func NewStartup(lc fx.Lifecycle) *Startup {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// log
			flog.Init(true, false)
			flog.Info("version %s %s", version.Buildtags, version.Buildstamp)

			// check singleton
			utils.CheckSingleton()

			// embed server
			utils.EmbedServer()
			return nil
		},
	})

	return &Startup{}
}
