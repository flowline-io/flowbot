package shell

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func skipIfWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows: requires bash and Unix paths")
	}
}

// mockReexec creates a simple exec.Cmd for testing without re-executing the binary
func mockReexec(args ...string) *exec.Cmd {
	// Skip the first argument (program name) and the -uid/-gid flags
	// Find where the actual shell command starts
	var cmdArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == "-uid" {
			i++ // skip the -uid value
		} else if args[i] == "-gid" {
			i++ // skip the -gid value
		} else if args[i] != "shell" {
			cmdArgs = append(cmdArgs, args[i])
		}
	}
	if len(cmdArgs) == 0 {
		cmdArgs = args
	}
	if len(cmdArgs) >= 2 {
		return exec.Command(cmdArgs[0], cmdArgs[1:]...)
	}
	return exec.Command("bash", "-c", "")
}

func TestShellRuntimeRunResult(t *testing.T) {
	skipIfWindows(t)
	rt := NewShellRuntime(Config{
		UID:   DefaultUid,
		GID:   DefaultGid,
		Rexec: mockReexec,
	})

	tk := &types.Task{
		ID:  utils.NewUUID(),
		Run: "echo -n hello world > $OUTPUT",
	}

	err := rt.Run(context.Background(), tk)

	assert.NoError(t, err)
	assert.Equal(t, "hello world", tk.Result)
}

func TestShellRuntimeRunFile(t *testing.T) {
	skipIfWindows(t)
	rt := NewShellRuntime(Config{
		UID:   DefaultUid,
		GID:   DefaultGid,
		Rexec: mockReexec,
	})

	tk := &types.Task{
		ID:  utils.NewUUID(),
		Run: "cat hello.txt > $OUTPUT",
		Files: map[string]string{
			"hello.txt": "hello world",
		},
	}

	err := rt.Run(context.Background(), tk)

	assert.NoError(t, err)
	assert.Equal(t, "hello world", tk.Result)
}

func TestShellRuntimeRunNotSupported(t *testing.T) {
	skipIfWindows(t)
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
	skipIfWindows(t)
	rt := NewShellRuntime(Config{
		UID:   DefaultUid,
		GID:   DefaultGid,
		Rexec: mockReexec,
	})

	tk := &types.Task{
		ID:  utils.NewUUID(),
		Run: "no_such_command",
	}

	err := rt.Run(context.Background(), tk)

	assert.Error(t, err)
}

func TestShellRuntimeRunTimeout(t *testing.T) {
	skipIfWindows(t)
	rt := NewShellRuntime(Config{
		UID:   DefaultUid,
		GID:   DefaultGid,
		Rexec: mockReexec,
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
	skipIfWindows(t)
	rt := NewShellRuntime(Config{
		UID:   DefaultUid,
		GID:   DefaultGid,
		Rexec: mockReexec,
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
