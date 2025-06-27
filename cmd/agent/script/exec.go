package script

import (
	"context"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/shell"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

func execScript(ctx context.Context, r Rule) (*types.Task, error) {
	rt := shell.NewShellRuntime(shell.Config{
		CMD: []string{"/bin/sh", "-c"},
		UID: config.App.ScriptEngine.UID,
		GID: config.App.ScriptEngine.GID,
	})

	task := &types.Task{
		ID:  utils.NewUUID(),
		Run: r.Path,
	}
	err := rt.Run(ctx, task)
	if err != nil {
		return nil, err
	}
	return task, nil
}
