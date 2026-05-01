package homelab

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

type DockerComposeRuntime struct {
	socketPath string
	appsDir    string
}

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
	if !strings.HasPrefix(absAppPath, absAppsDir) {
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
	composeFile := app.ComposeFile
	if composeFile == "" {
		composeFile = "docker-compose.yaml"
	}
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

	if strings.TrimSpace(output) == "" || output == "[]" || output == "null" {
		return AppStatusStopped, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return AppStatusStopped, nil
	}

	exitedCount := 0
	runningCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, `"exited"`) || strings.Contains(line, `"EXITED"`) {
			exitedCount++
		} else {
			runningCount++
		}
	}

	if runningCount > 0 && exitedCount == 0 {
		return AppStatusRunning, nil
	}
	if runningCount > 0 {
		return AppStatusPartial, nil
	}
	if exitedCount > 0 {
		return AppStatusStopped, nil
	}
	return AppStatusUnknown, nil
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
