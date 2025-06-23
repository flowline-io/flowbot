package script

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/shell"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

func (e *Engine) execWorker(r Rule) {
	rt := shell.NewShellRuntime(shell.Config{
		CMD: []string{"/bin/sh", "-c", r.Path},
		UID: config.App.ScriptEngine.UID,
		GID: config.App.ScriptEngine.GID,
	})
	if r.Timeout == 0 {
		r.Timeout = time.Hour
	}
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	task := &types.Task{
		ID: utils.NewUUID(),
	}
	err := rt.Run(ctx, task)
	if err != nil {
		flog.Error(err)
		return
	}

	select {
	case <-e.stop:
		flog.Info("cron script %s stopped", r.Id)
		err = rt.Stop(context.Background(), task)
		if err != nil {
			flog.Error(err)
		}
	case <-ctx.Done():
		flog.Info("exec script timout, %s, %s", r.Id, r.Path)
	}
}
