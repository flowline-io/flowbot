// Package env provides filesystem and shell execution with Result-based error handling.
package env

import (
	"context"
	"os"

	"github.com/flowline-io/flowbot/pkg/agent/result"
)

// Capture holds combined shell output from Exec.
type Capture struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecOptions configures a shell command invocation.
type ExecOptions struct {
	// Command runs through the platform shell when Argv is empty.
	Command string
	// Argv runs the binary directly without a shell when non-empty.
	Argv []string
	Dir  string
	// Timeout is the context governing cancellation and deadlines.
	Timeout context.Context
}

// DirEntry describes one filesystem entry from ReadDir.
type DirEntry struct {
	// Name is the entry basename.
	Name string
	// IsDir reports whether the entry is a directory.
	IsDir bool
}

// ExecutionEnv performs filesystem and shell operations without throwing untyped errors.
type ExecutionEnv interface {
	ReadFile(ctx context.Context, path string) result.Result[[]byte, result.FileError]
	WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) result.Result[struct{}, result.FileError]
	MkdirAll(ctx context.Context, path string, perm os.FileMode) result.Result[struct{}, result.FileError]
	Remove(ctx context.Context, path string) result.Result[struct{}, result.FileError]
	ReadDir(ctx context.Context, path string) result.Result[[]DirEntry, result.FileError]
	Exec(ctx context.Context, opts ExecOptions) result.Result[Capture, result.ExecutionError]
}

// Default returns the OS-backed execution environment.
func Default() ExecutionEnv {
	return OSExecutionEnv{}
}
