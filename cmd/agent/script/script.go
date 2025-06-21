package script

import (
	"context"
	"github.com/flc1125/go-cron/v4"
	"github.com/flowline-io/flowbot/cmd/agent/config"
	"go.uber.org/fx"
	"sync"
)

type Engine struct {
	c           *cron.Cron
	cronScripts sync.Map
	stop        chan struct{}
}

func NewEngine(lc fx.Lifecycle, _ config.Type) *Engine {
	e := &Engine{}

	if !config.App.ScriptEngine.Enabled {
		return e
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// scan scripts
			err := e.scan()
			if err != nil {
				return err
			}

			// run cron
			e.Cron()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			e.c.Stop()
			e.Shutdown()
			return nil
		},
	})

	return e
}

func (e *Engine) Shutdown() {
	e.stop <- struct{}{}
}
