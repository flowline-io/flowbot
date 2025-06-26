package startup

import (
	"context"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
	"go.uber.org/fx"
)

type Startup struct{}

func NewStartup(lc fx.Lifecycle, _ config.Type) *Startup {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// log
			flog.Init(false, false)
			flog.SetLevel(config.App.LogLevel)
			flog.Info("[version] %s %s", version.Buildtags, version.Buildstamp)

			// check singleton
			utils.CheckSingleton()

			// embed server
			go utils.EmbedServer()
			return nil
		},
	})

	return &Startup{}
}
