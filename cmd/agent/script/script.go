package script

import (
	"context"
	"sync"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"go.uber.org/fx"
)

type Engine struct {
	stop        chan struct{}
	cronScripts sync.Map
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

			go e.queue()
			go e.cron()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			e.Shutdown()
			return nil
		},
	})

	return e
}

func (e *Engine) Shutdown() {
	e.stop <- struct{}{}
}
