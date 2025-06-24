package script

import (
	"context"
	"database/sql"
	"runtime"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/cmd/agent/startup"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/riverqueue/river"
	"go.uber.org/fx"
)

type Engine struct {
	stop         chan struct{}
	queueStarted chan struct{}
	client       *river.Client[*sql.Tx]
}

func NewEngine(lc fx.Lifecycle, _ config.Type, _ *startup.Startup) *Engine {
	e := &Engine{
		stop:         make(chan struct{}),
		queueStarted: make(chan struct{}),
	}

	if !config.App.ScriptEngine.Enabled {
		return e
	}

	if runtime.GOOS != "linux" {
		return e
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go e.queue()

			<-e.queueStarted

			// scan scripts
			err := e.scan()
			if err != nil {
				return err
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Stop fetching new work and wait for active jobs to finish.
			ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			if err := e.client.Stop(ctx); err != nil {
				flog.Error(err)
			}
			e.Shutdown()
			return nil
		},
	})

	return e
}

func (e *Engine) Shutdown() {
	e.stop <- struct{}{}
	flog.Info("script engine shutdown")
}
