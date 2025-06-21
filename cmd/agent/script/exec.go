package script

import (
	"context"
	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/shell"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"time"
)

func (e *Engine) execWorker(r Rule) {
	// todo run script
	// todo timeout control

	rt := shell.NewShellRuntime(shell.Config{
		CMD: []string{"/bin/sh", "-c", r.Path},
		UID: config.App.ScriptEngine.UID,
		GID: config.App.ScriptEngine.GID,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
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
	case <-ctx.Done():
		flog.Info("timeout, kill exec process, %s, %s", r.Id, r.Path)
		err = rt.Stop(context.Background(), task)
		if err != nil {
			flog.Error(err)
			return
		}
		return
	}
}
