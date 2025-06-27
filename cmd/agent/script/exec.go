package script

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/adrg/xdg"
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

func onceLock(id string) (*os.File, error) {
	if xdg.ConfigHome == "" {
		return nil, fmt.Errorf("xdg.ConfigHome is empty")
	}
	lockPath := fmt.Sprintf("%s/flowbot/lock", xdg.ConfigHome)
	if err := os.MkdirAll(lockPath, 0700); err != nil {
		return nil, err
	}
	lockFilePath := fmt.Sprintf("%s/%s.lock", lockPath, utils.SHA1(id))

	file, err := os.OpenFile(lockFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = file.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, err
		}
		return nil, err
	}

	stat, _ := file.Stat()
	if stat.Size() > 0 {
		_ = file.Close()
		return nil, fmt.Errorf("lock file exist")
	}

	return file, nil
}
