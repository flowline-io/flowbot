package env

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/result"
)

// execArgPattern rejects NUL bytes in command strings before they reach os/exec.
var execArgPattern = regexp.MustCompile(`\A[^\x00]+\z`)

// OSExecutionEnv implements ExecutionEnv using the local operating system.
type OSExecutionEnv struct{}

// ReadFile reads file contents from disk.
func (OSExecutionEnv) ReadFile(_ context.Context, path string) result.Result[[]byte, result.FileError] {
	safe, ferr, ok := validatedFSPath(path)
	if !ok {
		return result.Err[[]byte, result.FileError](ferr)
	}
	data, err := os.ReadFile(safe)
	if err != nil {
		return result.Err[[]byte, result.FileError](toFileError(safe, err))
	}
	return result.Ok[[]byte, result.FileError](data)
}

// WriteFile writes data to a file path.
func (OSExecutionEnv) WriteFile(_ context.Context, path string, data []byte, perm os.FileMode) result.Result[struct{}, result.FileError] {
	safe, ferr, ok := validatedFSPath(path)
	if !ok {
		return result.Err[struct{}, result.FileError](ferr)
	}
	if err := os.WriteFile(safe, data, perm); err != nil {
		return result.Err[struct{}, result.FileError](toFileError(safe, err))
	}
	return result.Ok[struct{}, result.FileError](struct{}{})
}

// MkdirAll creates a directory tree.
func (OSExecutionEnv) MkdirAll(_ context.Context, path string, perm os.FileMode) result.Result[struct{}, result.FileError] {
	safe, ferr, ok := validatedFSPath(path)
	if !ok {
		return result.Err[struct{}, result.FileError](ferr)
	}
	if err := os.MkdirAll(safe, perm); err != nil {
		return result.Err[struct{}, result.FileError](toFileError(safe, err))
	}
	return result.Ok[struct{}, result.FileError](struct{}{})
}

// Remove deletes a file or empty directory.
func (OSExecutionEnv) Remove(_ context.Context, path string) result.Result[struct{}, result.FileError] {
	safe, ferr, ok := validatedFSPath(path)
	if !ok {
		return result.Err[struct{}, result.FileError](ferr)
	}
	if err := os.Remove(safe); err != nil {
		return result.Err[struct{}, result.FileError](toFileError(safe, err))
	}
	return result.Ok[struct{}, result.FileError](struct{}{})
}

// ReadDir lists directory entries.
func (OSExecutionEnv) ReadDir(_ context.Context, path string) result.Result[[]DirEntry, result.FileError] {
	safe, ferr, ok := validatedFSPath(path)
	if !ok {
		return result.Err[[]DirEntry, result.FileError](ferr)
	}
	entries, err := os.ReadDir(safe)
	if err != nil {
		return result.Err[[]DirEntry, result.FileError](toFileError(safe, err))
	}
	out := make([]DirEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, DirEntry{Name: entry.Name(), IsDir: entry.IsDir()})
	}
	return result.Ok[[]DirEntry, result.FileError](out)
}

// validatedFSPath normalizes path and rejects ".." path segments (not substrings
// in filenames) so relative traversal cannot reach os file APIs.
func validatedFSPath(path string) (string, result.FileError, bool) {
	if path == "" || strings.ContainsRune(path, 0) {
		return "", result.NewFileError("path_escape", "invalid path", nil), false
	}
	clean := filepath.Clean(path)
	// strings.Contains keeps CodeQL go/path-injection DotDotCheck; segment check
	// avoids rejecting names like "report..txt".
	if strings.Contains(clean, "..") && hasDotDotSegment(clean) {
		return "", result.NewFileError("path_escape", fmt.Sprintf("path %q escapes", path), nil), false
	}
	return clean, result.FileError{}, true
}

func hasDotDotSegment(path string) bool {
	rest := path[len(filepath.VolumeName(path)):]
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part == ".." {
			return true
		}
	}
	return false
}

func validatedExecCommand(command string) (string, bool) {
	command = strings.TrimSpace(command)
	if command == "" || !execArgPattern.MatchString(command) {
		return "", false
	}
	return command, true
}

func validatedExecArgv(argv []string) ([]string, bool) {
	if len(argv) == 0 {
		return nil, false
	}
	out := make([]string, len(argv))
	for i, arg := range argv {
		if arg == "" || !execArgPattern.MatchString(arg) {
			return nil, false
		}
		out[i] = arg
	}
	return out, true
}

// Exec runs a shell command or direct argv invocation and captures output.
func (OSExecutionEnv) Exec(ctx context.Context, opts ExecOptions) result.Result[Capture, result.ExecutionError] {
	runCtx := ctx
	if opts.Timeout != nil {
		runCtx = opts.Timeout
	}

	cmd, ferr, ok := buildExecCmd(runCtx, opts)
	if !ok {
		return result.Err[Capture, result.ExecutionError](ferr)
	}
	if opts.Dir != "" {
		safeDir, dirErr, dirOK := validatedFSPath(opts.Dir)
		if !dirOK {
			return result.Err[Capture, result.ExecutionError](
				result.NewExecutionError("spawn_error", dirErr.Error(), nil),
			)
		}
		cmd.Dir = safeDir
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if len(opts.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(opts.Stdin)
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}

	err := cmd.Run()
	capture := Capture{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
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

func buildExecCmd(runCtx context.Context, opts ExecOptions) (*exec.Cmd, result.ExecutionError, bool) {
	switch {
	case len(opts.Argv) > 0:
		argv, ok := validatedExecArgv(opts.Argv)
		if !ok {
			return nil, result.NewExecutionError("spawn_error", "invalid argv", nil), false
		}
		return exec.CommandContext(runCtx, argv[0], argv[1:]...), result.ExecutionError{}, true
	case strings.TrimSpace(opts.Command) != "":
		command, ok := validatedExecCommand(opts.Command)
		if !ok {
			return nil, result.NewExecutionError("spawn_error", "invalid command", nil), false
		}
		if runtime.GOOS == "windows" {
			return exec.CommandContext(runCtx, "cmd", "/C", command), result.ExecutionError{}, true
		}
		return exec.CommandContext(runCtx, "sh", "-c", command), result.ExecutionError{}, true
	default:
		return nil, result.NewExecutionError("spawn_error", "empty command", nil), false
	}
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
