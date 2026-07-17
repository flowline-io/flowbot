package env_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOSExecutionEnvReadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantOk   bool
		wantCode string
	}{
		{
			name: "reads existing file",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "hello.txt")
				require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))
				return path
			},
			wantOk: true,
		},
		{
			name: "not found",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing.txt")
			},
			wantOk:   false,
			wantCode: "not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := tt.setup(t)
			readResult := env.Default().ReadFile(context.Background(), path)
			assert.Equal(t, tt.wantOk, readResult.IsOk())
			if tt.wantOk {
				if tt.name == "reads existing file" {
					assert.Equal(t, "hello", string(readResult.Value()))
				}
				return
			}
			assert.Equal(t, tt.wantCode, readResult.ErrorValue().Code())
		})
	}
}

func TestOSExecutionEnvReadDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantOk   bool
		wantCode string
		wantLen  int
	}{
		{
			name: "lists directory entries",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644))
				require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
				return dir
			},
			wantOk:  true,
			wantLen: 2,
		},
		{
			name: "not found",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "missing")
			},
			wantOk:   false,
			wantCode: "not_found",
		},
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantOk:  true,
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := tt.setup(t)
			got := env.Default().ReadDir(context.Background(), path)
			assert.Equal(t, tt.wantOk, got.IsOk())
			if !tt.wantOk {
				assert.Equal(t, tt.wantCode, got.ErrorValue().Code())
				return
			}
			assert.Len(t, got.Value(), tt.wantLen)
		})
	}
}

func TestOSExecutionEnvExec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		command  string
		timeout  time.Duration
		wantOk   bool
		wantCode string
		wantExit int
	}{
		{
			name:     "successful command",
			command:  "echo ok",
			wantOk:   true,
			wantExit: 0,
		},
		{
			name:     "nonzero exit returns ok capture",
			command:  "exit 7",
			wantOk:   true,
			wantExit: 7,
		},
		{
			name:     "timeout",
			command:  slowCommand(),
			timeout:  50 * time.Millisecond,
			wantOk:   false,
			wantCode: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			var runCtx context.Context
			var cancel context.CancelFunc
			if tt.timeout > 0 {
				runCtx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else {
				runCtx = ctx
			}

			execResult := env.Default().Exec(runCtx, env.ExecOptions{
				Command: tt.command,
				Timeout: runCtx,
			})
			assert.Equal(t, tt.wantOk, execResult.IsOk())
			if !tt.wantOk {
				assert.Equal(t, tt.wantCode, execResult.ErrorValue().Code())
				return
			}
			assert.Equal(t, tt.wantExit, execResult.Value().ExitCode)
		})
	}
}

func TestFormatExecOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		capture env.Capture
		want    string
	}{
		{name: "stdout only", capture: env.Capture{Stdout: "hello"}, want: "hello"},
		{name: "nonzero exit", capture: env.Capture{Stdout: "fail", ExitCode: 2}, want: "exit code 2\nfail"},
		{name: "empty", capture: env.Capture{}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := env.FormatExecOutput(tt.capture, tt.capture.ExitCode != 0, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func slowCommand() string {
	if runtime.GOOS == "windows" {
		return "powershell -Command Start-Sleep -Seconds 5"
	}
	return "sleep 5"
}

func TestToFileErrorCodes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "not_found", result.NewFileError("not_found", "x", os.ErrNotExist).Code())
}
