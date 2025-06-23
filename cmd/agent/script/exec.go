package script

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/shell"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

func execScript(r Rule) error {
	rt := shell.NewShellRuntime(shell.Config{
		CMD: []string{"/bin/sh", "-c"},
		UID: config.App.ScriptEngine.UID,
		GID: config.App.ScriptEngine.GID,
	})
	if r.Timeout == 0 {
		r.Timeout = time.Hour
	}
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	task := &types.Task{
		ID:  utils.NewUUID(),
		Run: r.Path,
	}
	err := rt.Run(ctx, task)
	if err != nil {
		return err
	}
	return nil
}
