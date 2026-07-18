package homelab

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

type composePSEntry struct {
	State string `json:"State"`
}

// parseComposePSStatus parses docker compose ps --format json output into an AppStatus.
// Compose v2.21+ emits JSON Lines (one object per line); older versions emit a JSON array.
func parseComposePSStatus(output string) AppStatus {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return AppStatusStopped
	}

	entries, ok := parseComposePSEntries(trimmed)
	if !ok {
		return AppStatusUnknown
	}

	exitedCount := 0
	runningCount := 0
	for _, entry := range entries {
		switch strings.ToLower(entry.State) {
		case "exited", "dead":
			exitedCount++
		case "running":
			runningCount++
		}
	}

	if runningCount > 0 && exitedCount == 0 {
		return AppStatusRunning
	}
	if runningCount > 0 {
		return AppStatusPartial
	}
	if exitedCount > 0 {
		return AppStatusStopped
	}
	return AppStatusUnknown
}

// parseComposePSEntries decodes compose ps JSON array or JSON Lines into entries.
func parseComposePSEntries(trimmed string) ([]composePSEntry, bool) {
	if strings.HasPrefix(trimmed, "[") {
		var entries []composePSEntry
		if err := sonic.Unmarshal([]byte(trimmed), &entries); err != nil {
			return nil, false
		}
		return entries, true
	}

	lines := strings.Split(trimmed, "\n")
	entries := make([]composePSEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry composePSEntry
		if err := sonic.Unmarshal([]byte(line), &entry); err != nil {
			return nil, false
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, false
	}
	return entries, true
}

// DockerComposeRuntime executes docker compose commands locally using the
// Docker CLI against a configured Docker socket.
type DockerComposeRuntime struct {
	socketPath string
	appsDir    string
}

// NewDockerComposeRuntime creates a DockerComposeRuntime that shells out to
// docker compose using the configured socket path.
func NewDockerComposeRuntime(config RuntimeConfig, appsDir string) *DockerComposeRuntime {
	return &DockerComposeRuntime{
		socketPath: config.DockerSocket,
		appsDir:    appsDir,
	}
}

func (r *DockerComposeRuntime) validatePath(app App) error {
	absAppsDir, err := filepath.Abs(r.appsDir)
	if err != nil {
		return types.Errorf(types.ErrInternal, "homelab resolve apps_dir: %v", err)
	}
	absAppPath, err := filepath.Abs(app.Path)
	if err != nil {
		return types.Errorf(types.ErrInternal, "homelab resolve app path: %v", err)
	}
	if !isInside(absAppsDir, absAppPath) {
		return types.Errorf(types.ErrForbidden, "app path %s is outside apps_dir %s", app.Path, r.appsDir)
	}
	return nil
}

func (r *DockerComposeRuntime) composeEnv() []string {
	env := os.Environ()
	if r.socketPath != "" {
		env = append(env, "DOCKER_HOST="+r.socketPath)
	}
	return env
}

func (r *DockerComposeRuntime) composeCmd(ctx context.Context, app App, args ...string) *exec.Cmd {
	composeFile := composeFileName(app.ComposeFile)
	cmdArgs := []string{"compose", "-f", composeFile}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Dir = app.Path
	cmd.Env = r.composeEnv()
	return cmd
}

func (r *DockerComposeRuntime) runCmd(ctx context.Context, app App, args ...string) (string, error) {
	cmd := r.composeCmd(ctx, app, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func (r *DockerComposeRuntime) Status(ctx context.Context, app App) (AppStatus, error) {
	if err := ctx.Err(); err != nil {
		return AppStatusUnknown, types.WrapError(types.ErrTimeout, "homelab status canceled", err)
	}
	if err := r.validatePath(app); err != nil {
		return AppStatusUnknown, err
	}

	output, err := r.runCmd(ctx, app, "ps", "--format", "json")
	if err != nil {
		flog.Warn("docker compose ps failed for %s: %v", app.Name, err)
		return AppStatusUnknown, types.WrapError(types.ErrProvider, "docker compose ps", err)
	}

	return parseComposePSStatus(output), nil
}

func (r *DockerComposeRuntime) Logs(ctx context.Context, app App, tail int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "homelab logs canceled", err)
	}
	if err := r.validatePath(app); err != nil {
		return nil, err
	}

	output, err := r.runCmd(ctx, app, "logs", fmt.Sprintf("--tail=%d", tail))
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "docker compose logs", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

func (r *DockerComposeRuntime) Start(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab start canceled", err)
	}
	if err := r.validatePath(app); err != nil {
		return err
	}

	_, err := r.runCmd(ctx, app, "up", "-d")
	if err != nil {
		return types.WrapError(types.ErrProvider, "docker compose up", err)
	}
	return nil
}

func (r *DockerComposeRuntime) Stop(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab stop canceled", err)
	}
	if err := r.validatePath(app); err != nil {
		return err
	}

	_, err := r.runCmd(ctx, app, "down")
	if err != nil {
		return types.WrapError(types.ErrProvider, "docker compose down", err)
	}
	return nil
}

func (r *DockerComposeRuntime) Restart(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab restart canceled", err)
	}
	if err := r.validatePath(app); err != nil {
		return err
	}

	_, err := r.runCmd(ctx, app, "restart")
	if err != nil {
		return types.WrapError(types.ErrProvider, "docker compose restart", err)
	}
	return nil
}

func (r *DockerComposeRuntime) Pull(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab pull canceled", err)
	}
	if err := r.validatePath(app); err != nil {
		return err
	}

	_, err := r.runCmd(ctx, app, "pull")
	if err != nil {
		return types.WrapError(types.ErrProvider, "docker compose pull", err)
	}
	return nil
}

func (r *DockerComposeRuntime) Update(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab update canceled", err)
	}
	if err := r.validatePath(app); err != nil {
		return err
	}

	if _, err := r.runCmd(ctx, app, "pull"); err != nil {
		return types.WrapError(types.ErrProvider, "docker compose pull", err)
	}

	if _, err := r.runCmd(ctx, app, "up", "-d"); err != nil {
		return types.WrapError(types.ErrProvider, "docker compose up", err)
	}
	return nil
}
