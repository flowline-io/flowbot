package shell

import (
	"context"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"testing"
	"time"
)

func TestShellRuntimeRunResult(t *testing.T) {
	rt := NewShellRuntime(Config{
		UID: DEFAULT_UID,
		GID: DEFAULT_GID,
		Rexec: func(args ...string) *exec.Cmd {
			cmd := exec.Command(args[5], args[6:]...)
			return cmd
		},
	})

	tk := &types.Task{
		ID:  utils.NewUUID(),
		Run: "echo -n hello world > $REEXEC_FLOWBOT_OUTPUT",
	}

	err := rt.Run(context.Background(), tk)

	assert.NoError(t, err)
	assert.Equal(t, "hello world", tk.Result)
}

func TestShellRuntimeRunFile(t *testing.T) {
	rt := NewShellRuntime(Config{
		UID: DEFAULT_UID,
		GID: DEFAULT_GID,
		Rexec: func(args ...string) *exec.Cmd {
			cmd := exec.Command(args[5], args[6:]...)
			return cmd
		},
	})

	tk := &types.Task{
		ID:  utils.NewUUID(),
		Run: "cat hello.txt > $REEXEC_FLOWBOT_OUTPUT",
		Files: map[string]string{
			"hello.txt": "hello world",
		},
	}

	err := rt.Run(context.Background(), tk)

	assert.NoError(t, err)
	assert.Equal(t, "hello world", tk.Result)
}

func TestShellRuntimeRunNotSupported(t *testing.T) {
	rt := NewShellRuntime(Config{})

	tk := &types.Task{
		ID:       utils.NewUUID(),
		Run:      "echo hello world",
		Networks: []string{"some-network"},
	}

	err := rt.Run(context.Background(), tk)

	assert.Error(t, err)
}

func TestShellRuntimeRunError(t *testing.T) {
	rt := NewShellRuntime(Config{
		UID: DEFAULT_UID,
		GID: DEFAULT_GID,
		Rexec: func(args ...string) *exec.Cmd {
			cmd := exec.Command(args[5], args[6:]...)
			return cmd
		},
	})

	tk := &types.Task{
		ID:  utils.NewUUID(),
		Run: "no_such_command",
	}

	err := rt.Run(context.Background(), tk)

	assert.Error(t, err)
}

func TestShellRuntimeRunTimeout(t *testing.T) {
	rt := NewShellRuntime(Config{
		UID: DEFAULT_UID,
		GID: DEFAULT_GID,
		Rexec: func(args ...string) *exec.Cmd {
			cmd := exec.Command(args[5], args[6:]...)
			return cmd
		},
	})

	tk := &types.Task{
		ID:  utils.NewUUID(),
		Run: "sleep 30",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	err := rt.Run(ctx, tk)

	assert.Error(t, err)
}

func TestShellRuntimeStop(t *testing.T) {
	rt := NewShellRuntime(Config{
		UID: DEFAULT_UID,
		GID: DEFAULT_GID,
		Rexec: func(args ...string) *exec.Cmd {
			cmd := exec.Command(args[5], args[6:]...)
			return cmd
		},
	})

	tk := &types.Task{
		ID:  utils.NewUUID(),
		Run: "sleep 5",
	}

	ch := make(chan any)

	go func() {
		err := rt.Run(context.Background(), tk)
		assert.Error(t, err)
		close(ch)
	}()

	time.Sleep(time.Second * 1)

	err := rt.Stop(context.Background(), tk)
	assert.NoError(t, err)
	<-ch
}
