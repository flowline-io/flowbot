// Package exec provides shared workspace-bound terminal and code execution.
// It must not import pkg/capability (capability/core and agent tools call into this package).
package exec

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	// DefaultTimeout limits run_terminal and run_code when unset.
	DefaultTimeout = 60 * time.Second
	// MaxCodeBytes rejects run_code source above this size.
	MaxCodeBytes = 256 << 10
	// MaxJSONBytes is the function JSON in/out limit (64KiB).
	MaxJSONBytes = 64 << 10
	// DefaultMaxOutput truncates combined output beyond this byte count.
	DefaultMaxOutput = 8192

	goModContents = "module flowbotfn\n\ngo 1.26\n"
)

// Result holds process output from a terminal or code run.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Output   string
}

// Config bounds execution to a workspace root and an ExecutionEnv (OS or sandbox).
type Config struct {
	// Workspace is the absolute workspace root.
	Workspace string
	// Env performs filesystem and process operations; nil uses env.Default().
	Env env.ExecutionEnv
	// Timeout limits execution; zero uses DefaultTimeout.
	Timeout time.Duration
	// MaxOutput truncates Output; zero uses DefaultMaxOutput.
	MaxOutput int
}

func (c Config) executionEnv() env.ExecutionEnv {
	if c.Env != nil {
		return c.Env
	}
	return env.Default()
}

func (c Config) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return DefaultTimeout
}

func (c Config) maxOutput() int {
	if c.MaxOutput > 0 {
		return c.MaxOutput
	}
	return DefaultMaxOutput
}

// ResolveWorkDir returns an absolute directory under Workspace for relative workdir.
// Empty workdir uses the workspace root. Absolute workdir outside the root is rejected.
func (c Config) ResolveWorkDir(workdir string) (string, error) {
	root := strings.TrimSpace(c.Workspace)
	if root == "" {
		return "", fmt.Errorf("workspace is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	workdir = strings.TrimSpace(workdir)
	if workdir == "" {
		return absRoot, nil
	}
	if filepath.IsAbs(workdir) {
		return "", fmt.Errorf("workdir must be relative to workspace")
	}
	joined := filepath.Clean(filepath.Join(absRoot, workdir))
	rel, err := filepath.Rel(absRoot, joined)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("workdir %q escapes workspace", workdir)
	}
	return joined, nil
}

// RunTerminal executes a shell command in the workspace (optional relative workdir).
func RunTerminal(ctx context.Context, cfg Config, command, workdir string) (Result, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return Result{}, fmt.Errorf("command is required")
	}
	dir, err := cfg.ResolveWorkDir(workdir)
	if err != nil {
		return Result{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, cfg.timeout())
	defer cancel()
	execResult := cfg.executionEnv().Exec(runCtx, env.ExecOptions{
		Command: command,
		Dir:     dir,
		Timeout: runCtx,
	})
	if !execResult.IsOk() {
		return Result{}, fmt.Errorf("%s", env.FormatExecutionError(execResult.ErrorValue()))
	}
	capture := execResult.Value()
	return formatResult(capture, cfg.maxOutput()), nil
}

// RunCode writes source under .flowbot-run and invokes a language interpreter.
// stdin is optional process standard input passed through ExecOptions.
func RunCode(ctx context.Context, cfg Config, language, code, filename, workdir string, stdin []byte) (Result, error) {
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" || strings.TrimSpace(code) == "" {
		return Result{}, fmt.Errorf("language and code are required")
	}
	if len(code) > MaxCodeBytes {
		return Result{}, fmt.Errorf("code exceeds %d bytes", MaxCodeBytes)
	}
	dir, err := cfg.ResolveWorkDir(workdir)
	if err != nil {
		return Result{}, err
	}
	if filename == "" {
		filename = defaultFilename(language)
	}
	execEnv := cfg.executionEnv()
	scriptPath := filepath.Join(dir, ".flowbot-run", filepath.Base(filename))
	if mkdirResult := execEnv.MkdirAll(ctx, filepath.Dir(scriptPath), 0o755); !mkdirResult.IsOk() {
		return Result{}, fmt.Errorf("mkdir: %s", env.FormatFileError(mkdirResult.ErrorValue()))
	}
	if writeResult := execEnv.WriteFile(ctx, scriptPath, []byte(code), 0o644); !writeResult.IsOk() {
		return Result{}, fmt.Errorf("write code file: %s", env.FormatFileError(writeResult.ErrorValue()))
	}
	defer func() {
		if rem := execEnv.Remove(context.Background(), scriptPath); !rem.IsOk() {
			flog.Warn("exec: remove temp script %s: %s", scriptPath, env.FormatFileError(rem.ErrorValue()))
		}
	}()

	cmdArgs, err := interpreterCommand(language, scriptPath)
	if err != nil {
		return Result{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, cfg.timeout())
	defer cancel()
	execResult := execEnv.Exec(runCtx, env.ExecOptions{
		Argv:    cmdArgs,
		Dir:     dir,
		Stdin:   stdin,
		Timeout: runCtx,
	})
	if !execResult.IsOk() {
		return Result{}, fmt.Errorf("%s", env.FormatExecutionError(execResult.ErrorValue()))
	}
	return formatResult(execResult.Value(), cfg.maxOutput()), nil
}

// RunEntrypoint writes entrypoint at workspace root (main.py|main.sh|main.go), optional go.mod for go,
// and runs with stdin + env vars. It does not use .flowbot-run.
func RunEntrypoint(ctx context.Context, cfg Config, entrypoint, source string, stdin []byte, processEnv []string) (Result, error) {
	entrypoint = filepath.Base(strings.TrimSpace(entrypoint))
	switch entrypoint {
	case "main.py", "main.sh", "main.go":
	default:
		return Result{}, fmt.Errorf("entrypoint must be main.py, main.sh, or main.go")
	}
	if strings.TrimSpace(source) == "" {
		return Result{}, fmt.Errorf("source is required")
	}
	if len(source) > MaxCodeBytes {
		return Result{}, fmt.Errorf("source exceeds %d bytes", MaxCodeBytes)
	}
	language, err := languageFromEntrypoint(entrypoint)
	if err != nil {
		return Result{}, err
	}
	dir, err := cfg.ResolveWorkDir("")
	if err != nil {
		return Result{}, err
	}
	execEnv := cfg.executionEnv()
	scriptPath := filepath.Join(dir, entrypoint)
	if writeResult := execEnv.WriteFile(ctx, scriptPath, []byte(source), 0o644); !writeResult.IsOk() {
		return Result{}, fmt.Errorf("write entrypoint: %s", env.FormatFileError(writeResult.ErrorValue()))
	}
	envVars := append([]string(nil), processEnv...)
	if language == "go" {
		modPath := filepath.Join(dir, "go.mod")
		if writeResult := execEnv.WriteFile(ctx, modPath, []byte(goModContents), 0o644); !writeResult.IsOk() {
			return Result{}, fmt.Errorf("write go.mod: %s", env.FormatFileError(writeResult.ErrorValue()))
		}
		envVars = mergeGoEnv(dir, envVars)
	}
	cmdArgs, err := interpreterCommand(language, entrypoint)
	if err != nil {
		return Result{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, cfg.timeout())
	defer cancel()
	execResult := execEnv.Exec(runCtx, env.ExecOptions{
		Argv:    cmdArgs,
		Dir:     dir,
		Stdin:   stdin,
		Env:     envVars,
		Timeout: runCtx,
	})
	if !execResult.IsOk() {
		return Result{}, fmt.Errorf("%s", env.FormatExecutionError(execResult.ErrorValue()))
	}
	return formatResult(execResult.Value(), cfg.maxOutput()), nil
}

func languageFromEntrypoint(entrypoint string) (string, error) {
	switch entrypoint {
	case "main.py":
		return "python", nil
	case "main.sh":
		return "shell", nil
	case "main.go":
		return "go", nil
	default:
		return "", fmt.Errorf("unsupported entrypoint %q", entrypoint)
	}
}

func mergeGoEnv(workspace string, extra []string) []string {
	base := []string{
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOTELEMETRY=off",
		"GOTOOLCHAIN=local",
		"CGO_ENABLED=0",
		"GOCACHE=" + filepath.Join(workspace, ".gocache"),
		"GOPATH=" + filepath.Join(workspace, ".gopath"),
	}
	return append(base, extra...)
}

func formatResult(capture env.Capture, maxOutput int) Result {
	output := env.FormatExecOutput(capture, capture.ExitCode != 0, nil)
	if maxOutput > 0 && len(output) > maxOutput {
		output = output[:maxOutput] + "\n...[truncated]"
	}
	return Result{
		Stdout:   capture.Stdout,
		Stderr:   capture.Stderr,
		ExitCode: capture.ExitCode,
		Output:   output,
	}
}

func defaultFilename(language string) string {
	switch language {
	case "python", "py":
		return "script.py"
	case "shell", "sh", "bash":
		return "script.sh"
	case "go", "golang":
		return "main.go"
	default:
		return "snippet.txt"
	}
}

func interpreterCommand(language, filePath string) ([]string, error) {
	switch language {
	case "python", "py":
		return []string{"python", filePath}, nil
	case "shell", "sh", "bash":
		return []string{"sh", filePath}, nil
	case "go", "golang":
		return []string{"go", "run", filepath.Base(filePath)}, nil
	default:
		return nil, fmt.Errorf("unsupported language %q", language)
	}
}
