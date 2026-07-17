package sandbox_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/sandbox"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRunner struct {
	last sandbox.RunOptions
	cap  env.Capture
	err  error
}

func (m *mockRunner) Run(_ context.Context, opts sandbox.RunOptions) (env.Capture, error) {
	m.last = opts
	return m.cap, m.err
}

func TestConfigFromChatAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       config.ChatAgentSandboxConfig
		wantImage string
		wantURL   string
		wantToken string
	}{
		{name: "default image", cfg: config.ChatAgentSandboxConfig{}, wantImage: "ghcr.io/flowline-io/flowbot-agent-sandbox:latest"},
		{name: "custom image", cfg: config.ChatAgentSandboxConfig{Image: "custom:1"}, wantImage: "custom:1"},
		{name: "blank image uses default", cfg: config.ChatAgentSandboxConfig{Image: "  "}, wantImage: "ghcr.io/flowline-io/flowbot-agent-sandbox:latest"},
		{
			name: "cli credentials trimmed",
			cfg: config.ChatAgentSandboxConfig{
				Image:       "img:1",
				ServerURL:   "  http://host.docker.internal:6060  ",
				AccessToken: "  secret-token  ",
			},
			wantImage: "img:1",
			wantURL:   "http://host.docker.internal:6060",
			wantToken: "secret-token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sandbox.ConfigFromChatAgent(tt.cfg, "/ws")
			assert.Equal(t, tt.wantImage, got.Image)
			assert.Equal(t, "/ws", got.Workspace)
			assert.Equal(t, tt.wantURL, got.ServerURL)
			assert.Equal(t, tt.wantToken, got.AccessToken)
		})
	}
}

func TestEnvExecUsesRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		opts        env.ExecOptions
		runnerErr   error
		wantOK      bool
		wantCode    string
		wantCmd     string
		wantWorkDir string
	}{
		{
			name:        "command success",
			opts:        env.ExecOptions{Command: "echo hi", Dir: "/ws"},
			wantOK:      true,
			wantCmd:     "echo hi",
			wantWorkDir: "/ws",
		},
		{
			name:        "default workdir uses workspace",
			opts:        env.ExecOptions{Command: "pwd"},
			wantOK:      true,
			wantCmd:     "pwd",
			wantWorkDir: "/ws",
		},
		{
			name:        "subdir workdir under workspace",
			opts:        env.ExecOptions{Command: "pwd", Dir: "/ws/pkg"},
			wantOK:      true,
			wantCmd:     "pwd",
			wantWorkDir: "/ws/pkg",
		},
		{
			name:      "runner spawn error",
			opts:      env.ExecOptions{Command: "boom"},
			runnerErr: errors.New("docker down"),
			wantOK:    false,
			wantCode:  "spawn_error",
			wantCmd:   "boom",
		},
		{
			name:        "argv forwarded",
			opts:        env.ExecOptions{Argv: []string{"python", "x.py"}, Dir: "/ws"},
			wantOK:      true,
			wantWorkDir: "/ws",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := &mockRunner{
				cap: env.Capture{Stdout: "ok", ExitCode: 0},
				err: tt.runnerErr,
			}
			e := sandbox.New(sandbox.Config{
				Image:     "img:test",
				Workspace: "/ws",
			}, env.Default(), runner)
			got := e.Exec(context.Background(), tt.opts)
			assert.Equal(t, tt.wantOK, got.IsOk())
			assert.Equal(t, tt.wantCmd, runner.last.Command)
			if tt.wantWorkDir != "" {
				assert.Equal(t, tt.wantWorkDir, runner.last.WorkDir)
			}
			if len(tt.opts.Argv) > 0 {
				assert.Equal(t, tt.opts.Argv, runner.last.Argv)
			}
			if !tt.wantOK {
				assert.Equal(t, tt.wantCode, got.ErrorValue().Code())
			}
		})
	}
}

func TestEnvHostFilesystem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T, e *sandbox.Env, dir string)
	}{
		{
			name: "write and read",
			run: func(t *testing.T, e *sandbox.Env, dir string) {
				path := filepath.Join(dir, "a.txt")
				require.True(t, e.WriteFile(context.Background(), path, []byte("hi"), 0o644).IsOk())
				got := e.ReadFile(context.Background(), path)
				require.True(t, got.IsOk())
				assert.Equal(t, "hi", string(got.Value()))
			},
		},
		{
			name: "mkdir and remove",
			run: func(t *testing.T, e *sandbox.Env, dir string) {
				sub := filepath.Join(dir, "sub")
				require.True(t, e.MkdirAll(context.Background(), sub, 0o755).IsOk())
				file := filepath.Join(sub, "f.txt")
				require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
				require.True(t, e.Remove(context.Background(), file).IsOk())
			},
		},
		{
			name: "read missing",
			run: func(t *testing.T, e *sandbox.Env, dir string) {
				got := e.ReadFile(context.Background(), filepath.Join(dir, "missing.txt"))
				require.False(t, got.IsOk())
				assert.Equal(t, "not_found", got.ErrorValue().Code())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			e := sandbox.New(sandbox.Config{Workspace: dir, Image: "img"}, env.Default(), &mockRunner{})
			tt.run(t, e, dir)
		})
	}
}

func TestEnvExecCanceled(t *testing.T) {
	t.Parallel()
	runner := &mockRunner{err: context.Canceled}
	e := sandbox.New(sandbox.Config{Image: "img", Workspace: "/ws"}, env.Default(), runner)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := e.Exec(ctx, env.ExecOptions{Command: "sleep 1", Timeout: ctx})
	require.False(t, got.IsOk())
	assert.Equal(t, "aborted", got.ErrorValue().Code())
}

func TestEnvExecWorkdirEscape(t *testing.T) {
	t.Parallel()
	runner := &mockRunner{cap: env.Capture{ExitCode: 0}}
	e := sandbox.New(sandbox.Config{Image: "img", Workspace: "/ws"}, env.Default(), runner)
	got := e.Exec(context.Background(), env.ExecOptions{Command: "pwd", Dir: "/outside"})
	require.True(t, got.IsOk())
	assert.Equal(t, "/ws", runner.last.WorkDir)
}

func TestEnvExecTimeout(t *testing.T) {
	t.Parallel()
	runner := &mockRunner{err: context.DeadlineExceeded}
	e := sandbox.New(sandbox.Config{Image: "img", Workspace: "/ws"}, env.Default(), runner)
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	got := e.Exec(ctx, env.ExecOptions{Command: "sleep 1", Timeout: ctx})
	require.False(t, got.IsOk())
	assert.Equal(t, "timeout", got.ErrorValue().Code())
}

func TestEnvExecForwardsCLICredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serverURL   string
		accessToken string
	}{
		{name: "forwards both", serverURL: "http://host.docker.internal:6060", accessToken: "tok"},
		{name: "forwards empty when unset", serverURL: "", accessToken: ""},
		{name: "forwards token without url", serverURL: "", accessToken: "tok-only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := &mockRunner{cap: env.Capture{ExitCode: 0}}
			e := sandbox.New(sandbox.Config{
				Image:       "img",
				Workspace:   "/ws",
				ServerURL:   tt.serverURL,
				AccessToken: tt.accessToken,
			}, env.Default(), runner)
			got := e.Exec(context.Background(), env.ExecOptions{Command: "flowbot version"})
			require.True(t, got.IsOk())
			assert.Equal(t, tt.serverURL, runner.last.ServerURL)
			assert.Equal(t, tt.accessToken, runner.last.AccessToken)
		})
	}
}
