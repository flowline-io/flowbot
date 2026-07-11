// Package sandbox provides optional Docker isolation for agent shell and code tools.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	defaultImage        = "ghcr.io/flowline-io/flowbot-agent-sandbox:latest"
	workspaceMount      = "/workspace"
	defaultStopWait     = 5 * time.Second
	maxLoggedCommandLen = 200
)

// Config configures Docker sandbox execution.
type Config struct {
	// Image is the container image used for Exec.
	Image string
	// Network is the Docker network mode.
	Network string
	// Memory limits container memory (e.g. "512m").
	Memory string
	// Workspace is the host workspace path mounted at /workspace.
	Workspace string
}

// ConfigFromChatAgent builds sandbox Config from chat agent settings.
func ConfigFromChatAgent(cfg config.ChatAgentSandboxConfig, workspace string) Config {
	image := strings.TrimSpace(cfg.Image)
	if image == "" {
		image = defaultImage
	}
	return Config{
		Image:     image,
		Network:   strings.TrimSpace(cfg.Network),
		Memory:    strings.TrimSpace(cfg.Memory),
		Workspace: strings.TrimSpace(workspace),
	}
}

// Runner executes a one-shot command inside a sandbox container.
type Runner interface {
	Run(ctx context.Context, opts RunOptions) (env.Capture, error)
}

// RunOptions configures one sandbox command invocation.
type RunOptions struct {
	Image     string
	Network   string
	Memory    string
	Workspace string
	WorkDir   string
	Command   string
	Argv      []string
}

// Env implements env.ExecutionEnv with host filesystem ops and sandboxed Exec.
type Env struct {
	cfg    Config
	host   env.ExecutionEnv
	runner Runner
}

// New creates a sandbox ExecutionEnv. Host FS ops use env.Default when host is nil.
func New(cfg Config, host env.ExecutionEnv, runner Runner) *Env {
	if host == nil {
		host = env.Default()
	}
	if runner == nil {
		runner = DockerRunner{}
	}
	flog.Info("[sandbox] env ready workspace=%s image=%s network=%s memory=%s",
		cfg.Workspace, cfg.Image, cfg.Network, cfg.Memory)
	return &Env{cfg: cfg, host: host, runner: runner}
}

// ReadFile reads from the host filesystem.
func (e *Env) ReadFile(ctx context.Context, path string) result.Result[[]byte, result.FileError] {
	return e.host.ReadFile(ctx, path)
}

// WriteFile writes to the host filesystem.
func (e *Env) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) result.Result[struct{}, result.FileError] {
	return e.host.WriteFile(ctx, path, data, perm)
}

// MkdirAll creates directories on the host filesystem.
func (e *Env) MkdirAll(ctx context.Context, path string, perm os.FileMode) result.Result[struct{}, result.FileError] {
	return e.host.MkdirAll(ctx, path, perm)
}

// Remove deletes a path on the host filesystem.
func (e *Env) Remove(ctx context.Context, path string) result.Result[struct{}, result.FileError] {
	return e.host.Remove(ctx, path)
}

// Exec runs the command inside a sandbox container with the workspace mounted.
func (e *Env) Exec(ctx context.Context, opts env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	runCtx := ctx
	if opts.Timeout != nil {
		runCtx = opts.Timeout
	}
	workDir := workspaceMount
	if opts.Dir != "" && e.cfg.Workspace != "" {
		rel, err := filepath.Rel(e.cfg.Workspace, opts.Dir)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			workDir = filepath.ToSlash(filepath.Join(workspaceMount, rel))
		}
	}
	runOpts := RunOptions{
		Image:     e.cfg.Image,
		Network:   e.cfg.Network,
		Memory:    e.cfg.Memory,
		Workspace: e.cfg.Workspace,
		WorkDir:   workDir,
		Command:   opts.Command,
		Argv:      append([]string(nil), opts.Argv...),
	}
	flog.Info("[sandbox] exec start workspace=%s workdir=%s %s",
		e.cfg.Workspace, workDir, summarizeCommand(runOpts))
	capture, err := e.runner.Run(runCtx, runOpts)
	if err != nil {
		code := "spawn_error"
		cause := err
		switch {
		case runCtx.Err() == context.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded):
			code = "timeout"
			if runCtx.Err() != nil {
				cause = runCtx.Err()
			}
		case runCtx.Err() == context.Canceled || errors.Is(err, context.Canceled):
			code = "aborted"
			if runCtx.Err() != nil {
				cause = runCtx.Err()
			}
		}
		flog.Info("[sandbox] exec failed workspace=%s workdir=%s code=%s err=%s",
			e.cfg.Workspace, workDir, code, cause.Error())
		return result.Err[env.Capture, result.ExecutionError](
			result.NewExecutionError(code, cause.Error(), cause),
		)
	}
	flog.Info("[sandbox] exec done workspace=%s workdir=%s exit_code=%d",
		e.cfg.Workspace, workDir, capture.ExitCode)
	return result.Ok[env.Capture, result.ExecutionError](capture)
}

// DockerRunner runs commands via the Docker Engine API.
type DockerRunner struct{}

// Run starts an ephemeral container, waits for exit, and returns captured output.
func (DockerRunner) Run(ctx context.Context, opts RunOptions) (env.Capture, error) {
	if err := validateRunOptions(opts); err != nil {
		return env.Capture{}, err
	}
	cmd, err := buildCommand(opts)
	if err != nil {
		return env.Capture{}, err
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return env.Capture{}, err
	}
	defer cli.Close()

	hostConfig, err := buildHostConfig(opts)
	if err != nil {
		return env.Capture{}, err
	}
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = workspaceMount
	}
	flog.Info("[sandbox] container create image=%s workspace=%s workdir=%s %s",
		opts.Image, opts.Workspace, workDir, summarizeCommand(opts))
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      opts.Image,
		Cmd:        cmd,
		WorkingDir: workDir,
		Tty:        false,
	}, hostConfig, nil, nil, "")
	if err != nil {
		flog.Info("[sandbox] container create failed image=%s workspace=%s err=%s",
			opts.Image, opts.Workspace, err.Error())
		return env.Capture{}, err
	}
	id := resp.ID
	flog.Info("[sandbox] container created id=%s image=%s workdir=%s", id, opts.Image, workDir)
	defer func() {
		_ = cli.ContainerRemove(context.Background(), id, container.RemoveOptions{Force: true})
	}()
	return waitAndCollectLogs(ctx, cli, id, opts.Workspace, workDir)
}

func validateRunOptions(opts RunOptions) error {
	if strings.TrimSpace(opts.Workspace) == "" {
		return fmt.Errorf("sandbox: workspace is required")
	}
	if strings.TrimSpace(opts.Image) == "" {
		return fmt.Errorf("sandbox: image is required")
	}
	return nil
}

func buildHostConfig(opts RunOptions) (*container.HostConfig, error) {
	hostConfig := &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:%s", opts.Workspace, workspaceMount)},
	}
	if opts.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(opts.Network)
	}
	if opts.Memory != "" {
		n, parseErr := units.RAMInBytes(opts.Memory)
		if parseErr != nil {
			return nil, fmt.Errorf("sandbox: memory: %w", parseErr)
		}
		hostConfig.Resources.Memory = n
	}
	return hostConfig, nil
}

func waitAndCollectLogs(ctx context.Context, cli *client.Client, id, workspace, workDir string) (env.Capture, error) {
	if err := cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return env.Capture{}, err
	}
	statusCh, errCh := cli.ContainerWait(ctx, id, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case err := <-errCh:
		if err != nil {
			return env.Capture{}, err
		}
	case status := <-statusCh:
		exitCode = status.StatusCode
	case <-ctx.Done():
		stopCtx, cancel := context.WithTimeout(context.Background(), defaultStopWait)
		defer cancel()
		_ = cli.ContainerRemove(stopCtx, id, container.RemoveOptions{Force: true})
		return env.Capture{}, ctx.Err()
	}
	logs, err := cli.ContainerLogs(ctx, id, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return env.Capture{}, err
	}
	defer logs.Close()
	output, err := io.ReadAll(logs)
	if err != nil {
		return env.Capture{}, err
	}
	text := stripDockerLogHeaders(output)
	flog.Info("[sandbox] container done id=%s workspace=%s workdir=%s exit_code=%d",
		id, workspace, workDir, exitCode)
	return env.Capture{Stdout: text, Stderr: text, ExitCode: int(exitCode)}, nil
}

func summarizeCommand(opts RunOptions) string {
	if len(opts.Argv) > 0 {
		return "argv=" + truncateForLog(strings.Join(opts.Argv, " "))
	}
	return "command=" + truncateForLog(strings.TrimSpace(opts.Command))
}

func truncateForLog(text string) string {
	if len(text) <= maxLoggedCommandLen {
		return text
	}
	return text[:maxLoggedCommandLen] + "..."
}

func buildCommand(opts RunOptions) ([]string, error) {
	if len(opts.Argv) > 0 {
		return append([]string(nil), opts.Argv...), nil
	}
	if strings.TrimSpace(opts.Command) == "" {
		return nil, fmt.Errorf("sandbox: empty command")
	}
	return []string{"sh", "-c", opts.Command}, nil
}

func stripDockerLogHeaders(data []byte) string {
	if len(data) < 8 {
		return string(data)
	}
	var out bytes.Buffer
	rest := data
	wrote := false
	for len(rest) >= 8 {
		size := int(rest[4])<<24 | int(rest[5])<<16 | int(rest[6])<<8 | int(rest[7])
		rest = rest[8:]
		if size > len(rest) {
			_, _ = out.Write(rest)
			wrote = true
			break
		}
		_, _ = out.Write(rest[:size])
		wrote = true
		rest = rest[size:]
	}
	if !wrote {
		return string(data)
	}
	return out.String()
}

// Ensure Env implements ExecutionEnv.
var _ env.ExecutionEnv = (*Env)(nil)
