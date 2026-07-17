package env

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/result"
)

// OSExecutionEnv implements ExecutionEnv using the local operating system.
type OSExecutionEnv struct{}

// ReadFile reads file contents from disk.
func (OSExecutionEnv) ReadFile(_ context.Context, path string) result.Result[[]byte, result.FileError] {
	data, err := os.ReadFile(path)
	if err != nil {
		return result.Err[[]byte, result.FileError](toFileError(path, err))
	}
	return result.Ok[[]byte, result.FileError](data)
}

// WriteFile writes data to a file path.
func (OSExecutionEnv) WriteFile(_ context.Context, path string, data []byte, perm os.FileMode) result.Result[struct{}, result.FileError] {
	if err := os.WriteFile(path, data, perm); err != nil {
		return result.Err[struct{}, result.FileError](toFileError(path, err))
	}
	return result.Ok[struct{}, result.FileError](struct{}{})
}

// MkdirAll creates a directory tree.
func (OSExecutionEnv) MkdirAll(_ context.Context, path string, perm os.FileMode) result.Result[struct{}, result.FileError] {
	if err := os.MkdirAll(path, perm); err != nil {
		return result.Err[struct{}, result.FileError](toFileError(path, err))
	}
	return result.Ok[struct{}, result.FileError](struct{}{})
}

// Remove deletes a file or empty directory.
func (OSExecutionEnv) Remove(_ context.Context, path string) result.Result[struct{}, result.FileError] {
	if err := os.Remove(path); err != nil {
		return result.Err[struct{}, result.FileError](toFileError(path, err))
	}
	return result.Ok[struct{}, result.FileError](struct{}{})
}

// ReadDir lists directory entries.
func (OSExecutionEnv) ReadDir(_ context.Context, path string) result.Result[[]DirEntry, result.FileError] {
	entries, err := os.ReadDir(path)
	if err != nil {
		return result.Err[[]DirEntry, result.FileError](toFileError(path, err))
	}
	out := make([]DirEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, DirEntry{Name: entry.Name(), IsDir: entry.IsDir()})
	}
	return result.Ok[[]DirEntry, result.FileError](out)
}

// Exec runs a shell command or direct argv invocation and captures output.
func (OSExecutionEnv) Exec(ctx context.Context, opts ExecOptions) result.Result[Capture, result.ExecutionError] {
	runCtx := ctx
	if opts.Timeout != nil {
		runCtx = opts.Timeout
	}

	var cmd *exec.Cmd
	switch {
	case len(opts.Argv) > 0:
		cmd = exec.CommandContext(runCtx, opts.Argv[0], opts.Argv[1:]...)
	case strings.TrimSpace(opts.Command) != "":
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(runCtx, "cmd", "/C", opts.Command)
		} else {
			cmd = exec.CommandContext(runCtx, "sh", "-c", opts.Command)
		}
	default:
		return result.Err[Capture, result.ExecutionError](
			result.NewExecutionError("spawn_error", "empty command", nil),
		)
	}
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	capture := Capture{
		Stdout:   buf.String(),
		Stderr:   buf.String(),
		ExitCode: 0,
	}
	if runCtx.Err() != nil {
		code := "aborted"
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			code = "timeout"
		}
		return result.Err[Capture, result.ExecutionError](
			result.NewExecutionError(code, runCtx.Err().Error(), runCtx.Err()),
		)
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			capture.ExitCode = exitErr.ExitCode()
			return result.Ok[Capture, result.ExecutionError](capture)
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			code := "aborted"
			if errors.Is(err, context.DeadlineExceeded) {
				code = "timeout"
			}
			return result.Err[Capture, result.ExecutionError](
				result.NewExecutionError(code, err.Error(), err),
			)
		}
		return result.Err[Capture, result.ExecutionError](
			result.NewExecutionError("spawn_error", err.Error(), err),
		)
	}
	return result.Ok[Capture, result.ExecutionError](capture)
}

func toFileError(path string, err error) result.FileError {
	if os.IsNotExist(err) {
		return result.NewFileError("not_found", path, err)
	}
	if os.IsPermission(err) {
		return result.NewFileError("permission_denied", path, err)
	}
	return result.NewFileError("io_error", path, err)
}

// FormatFileError returns a tool-facing message for a FileError.
func FormatFileError(err result.FileError) string {
	return err.Error()
}

// FormatExecutionError returns a tool-facing message for an ExecutionError.
func FormatExecutionError(err result.ExecutionError) string {
	return err.Error()
}

// FormatExecOutput formats capture output for tool results.
func FormatExecOutput(capture Capture, isError bool, err error) string {
	output := strings.TrimSpace(capture.Stdout)
	if capture.Stderr != "" && capture.Stderr != capture.Stdout {
		if output != "" {
			output += "\n"
		}
		output += strings.TrimSpace(capture.Stderr)
	}
	if isError && err != nil {
		return strings.TrimSpace(fmt.Sprintf("exit error: %v\n%s", err, output))
	}
	if capture.ExitCode != 0 {
		return strings.TrimSpace(fmt.Sprintf("exit code %d\n%s", capture.ExitCode, output))
	}
	return output
}
