package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRunOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    RunOptions
		wantErr string
	}{
		{name: "valid options", opts: RunOptions{Workspace: "/ws", Image: "img:1"}},
		{name: "missing workspace", opts: RunOptions{Image: "img:1"}, wantErr: "workspace is required"},
		{name: "missing image", opts: RunOptions{Workspace: "/ws"}, wantErr: "image is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRunOptions(tt.opts)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestBuildCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    RunOptions
		want    []string
		wantErr string
	}{
		{name: "argv passthrough", opts: RunOptions{Argv: []string{"python", "main.py"}}, want: []string{"python", "main.py"}},
		{name: "shell wraps command", opts: RunOptions{Command: "echo hi"}, want: []string{"sh", "-c", "echo hi"}},
		{name: "empty command errors", opts: RunOptions{}, wantErr: "empty command"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := buildCommand(tt.opts)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripDockerLogHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{name: "short payload unchanged", in: []byte("hi"), want: "hi"},
		{name: "empty payload", in: []byte{}, want: ""},
		{name: "framed docker logs stripped", in: append([]byte{1, 0, 0, 0, 0, 0, 0, 3}, []byte("abc")...), want: "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stripDockerLogHeaders(tt.in))
		})
	}
}

func TestBuildHostConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		opts           RunOptions
		wantErr        string
		wantBinds      int
		wantExtraHosts bool
		wantCLIBind    bool
	}{
		{name: "binds workspace", opts: RunOptions{Workspace: "/host/ws"}, wantBinds: 1},
		{name: "sets network mode", opts: RunOptions{Workspace: "/host/ws", Network: "bridge"}, wantBinds: 1},
		{name: "invalid memory", opts: RunOptions{Workspace: "/host/ws", Memory: "not-memory"}, wantErr: "memory"},
		{
			name: "cli config bind and host gateway",
			opts: RunOptions{
				Workspace:    "/host/ws",
				CLIConfigDir: "/tmp/cli-cfg",
				ServerURL:    "http://host.docker.internal:6060",
			},
			wantBinds:      2,
			wantExtraHosts: true,
			wantCLIBind:    true,
		},
		{
			name: "no host gateway for other urls",
			opts: RunOptions{
				Workspace: "/host/ws",
				ServerURL: "http://flowbot:6060",
			},
			wantBinds:      1,
			wantExtraHosts: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hc, err := buildHostConfig(tt.opts)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, hc.Binds, tt.wantBinds)
			if tt.wantExtraHosts {
				require.Equal(t, []string{"host.docker.internal:host-gateway"}, hc.ExtraHosts)
			} else {
				assert.Empty(t, hc.ExtraHosts)
			}
			if tt.wantCLIBind {
				assert.Contains(t, hc.Binds[1], containerCLIConfigPath+":ro")
			}
		})
	}
}

func TestMaterializeCLIConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		serverURL string
		token     string
		wantURL   bool
		wantToken bool
	}{
		{name: "writes both files", serverURL: "http://host.docker.internal:6060", token: "tok", wantURL: true, wantToken: true},
		{name: "token only", serverURL: "", token: "tok", wantURL: false, wantToken: true},
		{name: "url only", serverURL: "http://flowbot:6060", token: "", wantURL: true, wantToken: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, err := materializeCLIConfig(tt.serverURL, tt.token)
			require.NoError(t, err)
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			assertAgentReadable(t, dir)

			tokenPath := filepath.Join(dir, cliTokenFileName)
			urlPath := filepath.Join(dir, cliServerURLFileName)
			if tt.wantToken {
				data, readErr := os.ReadFile(tokenPath)
				require.NoError(t, readErr)
				assert.Equal(t, tt.token, string(data))
				assertAgentReadable(t, tokenPath)
			} else {
				_, readErr := os.Stat(tokenPath)
				require.Error(t, readErr)
			}
			if tt.wantURL {
				data, readErr := os.ReadFile(urlPath)
				require.NoError(t, readErr)
				assert.Equal(t, tt.serverURL, string(data))
				assertAgentReadable(t, urlPath)
			} else {
				_, readErr := os.Stat(urlPath)
				require.Error(t, readErr)
			}
		})
	}
}

func assertAgentReadable(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	perm := info.Mode().Perm()
	if info.IsDir() {
		// Directories need an execute bit to be traversable; owner-only or world-accessible.
		assert.True(t, perm == cliConfigDirOwnerOnly || perm&0o001 != 0 || perm&0o010 != 0,
			"dir %s mode %o should be traversable by sandbox agent", path, perm)
		return
	}
	// After ensureSandboxAgentReadable: either owner-only (chown succeeded) or world-readable fallback.
	assert.True(t, perm == cliConfigOwnerOnly || perm&0o004 != 0 || perm&0o040 != 0,
		"path %s mode %o should be readable by sandbox agent", path, perm)
}

func TestEnsureSandboxAgentReadable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "file becomes readable after ensure"},
		{name: "directory remains traversable after ensure"},
		{name: "idempotent second ensure"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "secret")
			var nested string
			if strings.Contains(tt.name, "directory") {
				path = filepath.Join(dir, "subdir")
				require.NoError(t, os.Mkdir(path, 0o700))
				nested = filepath.Join(path, "nested")
				require.NoError(t, os.WriteFile(nested, []byte("x"), 0o600))
			} else {
				require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
			}
			require.NoError(t, ensureSandboxAgentReadable(path))
			assertAgentReadable(t, path)
			if nested != "" {
				data, readErr := os.ReadFile(nested)
				require.NoError(t, readErr, "directory must stay traversable after ensure")
				assert.Equal(t, "x", string(data))
			}
			if strings.Contains(tt.name, "idempotent") {
				require.NoError(t, ensureSandboxAgentReadable(path))
				assertAgentReadable(t, path)
			}
		})
	}
}

func TestBuildContainerEnv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts RunOptions
		want []string
	}{
		{name: "empty without token", opts: RunOptions{ServerURL: "http://x"}, want: nil},
		{
			name: "token and url",
			opts: RunOptions{ServerURL: "http://host.docker.internal:6060", AccessToken: "tok"},
			want: []string{"FLOWBOT_SERVER_URL=http://host.docker.internal:6060", "FLOWBOT_TOKEN=tok"},
		},
		{
			name: "token only",
			opts: RunOptions{AccessToken: "tok"},
			want: []string{"FLOWBOT_TOKEN=tok"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, buildContainerEnv(tt.opts))
		})
	}
}

func TestNeedsHostGateway(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "host docker internal", url: "http://host.docker.internal:6060", want: true},
		{name: "service name", url: "http://flowbot:6060", want: false},
		{name: "empty", url: "", want: false},
		{name: "localhost", url: "http://127.0.0.1:6060", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, needsHostGateway(tt.url))
		})
	}
}
