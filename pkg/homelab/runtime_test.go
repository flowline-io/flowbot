package homelab

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func generateTestHostKey(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(key)
	require.NoError(t, err)
	return string(ssh.MarshalAuthorizedKey(signer.PublicKey()))
}

func TestDockerComposeRuntime_ValidatePath(t *testing.T) {
	tmpDir := t.TempDir()
	appsDir := filepath.Join(tmpDir, "apps")
	require.NoError(t, os.MkdirAll(appsDir, 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(appsDir, "myapp"), 0700))
	outsideDir := filepath.Join(tmpDir, "outside")
	require.NoError(t, os.MkdirAll(outsideDir, 0700))

	r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket}, appsDir)

	tests := []struct {
		name    string
		app     App
		wantErr bool
	}{
		{
			name:    "valid path inside apps_dir",
			app:     App{Path: filepath.Join(appsDir, "myapp")},
			wantErr: false,
		},
		{
			name:    "path outside apps_dir is rejected",
			app:     App{Path: tmpDir},
			wantErr: true,
		},
		{
			name:    "path outside apps_dir via parent",
			app:     App{Path: outsideDir},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.validatePath(tt.app)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDockerComposeRuntime_ComposeEnv(t *testing.T) {
	tests := []struct {
		name   string
		socket string
	}{
		{name: "DOCKER_HOST included when socket set", socket: "unix:///var/run/docker.sock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket, DockerSocket: tt.socket}, "/apps")
			env := r.composeEnv()
			assert.Contains(t, env, "DOCKER_HOST=unix:///var/run/docker.sock")
		})
	}
}

func TestDockerComposeRuntime_ComposeEnvNoSocket(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "no DOCKER_HOST when socket is empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket}, "/apps")
			env := r.composeEnv()
			found := false
			for _, e := range env {
				if e == "DOCKER_HOST=" {
					found = true
				}
			}
			assert.False(t, found)
		})
	}
}

func TestDockerComposeRuntime_StatusNoDocker(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "status call without docker daemon"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			appDir := filepath.Join(tmpDir, "testapp")
			require.NoError(t, os.MkdirAll(appDir, 0700))

			r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket}, tmpDir)
			app := App{Name: "testapp", Path: appDir, ComposeFile: "docker-compose.yaml"}

			status, err := r.Status(t.Context(), app)
			if err != nil {
				t.Logf("expected error (no docker daemon): %v", err)
			} else {
				t.Logf("status: %s", status)
			}
		})
	}
}

func TestNoopRuntime_AllOperations(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "noop runtime operations"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NoopRuntime{}
			ctx := t.Context()
			app := App{Name: "testapp", Path: "/fake/path", Status: AppStatusUnknown}

			status, err := r.Status(ctx, app)
			assert.NoError(t, err)
			assert.Equal(t, AppStatusUnknown, status)

			_, err = r.Logs(ctx, app, 100)
			assert.Error(t, err)

			assert.Error(t, r.Start(ctx, app))
			assert.Error(t, r.Stop(ctx, app))
			assert.Error(t, r.Restart(ctx, app))
			assert.Error(t, r.Pull(ctx, app))
			assert.Error(t, r.Update(ctx, app))
		})
	}
}

func TestNewRuntime_ReturnsCorrectType(t *testing.T) {
	tests := []struct {
		mode     RuntimeMode
		wantType string
	}{
		{RuntimeModeNone, "*homelab.NoopRuntime"},
		{RuntimeModeDockerSocket, "*homelab.DockerComposeRuntime"},
		{RuntimeModeSSH, "*homelab.SSHRuntime"},
		{"unknown", "*homelab.NoopRuntime"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			config := RuntimeConfig{Mode: tt.mode}
			rt := NewRuntime(config, "/apps")
			assert.Equal(t, tt.wantType, getRuntimeTypeName(rt))
		})
	}
}

func getRuntimeTypeName(rt Runtime) string {
	switch rt.(type) {
	case NoopRuntime:
		return "*homelab.NoopRuntime"
	case *DockerComposeRuntime:
		return "*homelab.DockerComposeRuntime"
	case *SSHRuntime:
		return "*homelab.SSHRuntime"
	default:
		return "unknown"
	}
}

func TestSSHRuntime_ConfigDefaults(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "SSH runtime config defaults"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: "test",
				SSHHostKey:  generateTestHostKey(t),
			})
			assert.Equal(t, "example.com", r.host)
			assert.Equal(t, 22, r.port)
			assert.Equal(t, "root", r.user)
		})
	}
}

func TestSSHRuntime_ClientConfigNoAuth(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "client config fails without auth"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSSHRuntime(RuntimeConfig{
				Mode:       RuntimeModeSSH,
				SSHHost:    "example.com",
				SSHUser:    "root",
				SSHHostKey: generateTestHostKey(t),
			})
			_, err := r.clientConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "key or password")
		})
	}
}

func TestSSHRuntime_ClientConfigPasswordAuth(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "client config with password auth"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: "test",
				SSHHostKey:  generateTestHostKey(t),
			})
			cfg, err := r.clientConfig()
			require.NoError(t, err)
			assert.Equal(t, "root", cfg.User)
			require.Len(t, cfg.Auth, 1)
		})
	}
}

func TestSSHRuntime_DefaultPort(t *testing.T) {
	tests := []struct {
		name     string
		sshPort  int
		wantPort int
	}{
		{name: "zero port defaults to 22", sshPort: 0, wantPort: 22},
		{name: "custom port is used", sshPort: 2222, wantPort: 2222},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostKey := generateTestHostKey(t)
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: "test",
				SSHPort:     tt.sshPort,
				SSHHostKey:  hostKey,
			})
			assert.Equal(t, tt.wantPort, r.port)
		})
	}
}

func TestSSHRuntime_ContextCancellation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cancelled context causes errors for all operations"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: "test",
				SSHHostKey:  generateTestHostKey(t),
			})
			app := App{Name: "test", Path: "/test"}

			_, err := r.Status(ctx, app)
			require.Error(t, err)

			_, err = r.Logs(ctx, app, 10)
			require.Error(t, err)

			require.Error(t, r.Start(ctx, app))
			require.Error(t, r.Stop(ctx, app))
			require.Error(t, r.Restart(ctx, app))
			require.Error(t, r.Pull(ctx, app))
			require.Error(t, r.Update(ctx, app))
		})
	}
}

func TestSSHRuntime_ClientConfigNoHostKey(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "client config fails without host key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: "test",
			})
			_, err := r.clientConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "host_key")
		})
	}
}

func TestSSHRuntime_ClientConfigInvalidHostKey(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "client config fails with invalid host key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: "test",
				SSHHostKey:  "not-a-valid-key",
			})
			_, err := r.clientConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "parse ssh host key")
		})
	}
}

func TestSSHRuntime_ClientConfigFixedHostKey(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "fixed host key sets HostKeyCallback"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostKey := generateTestHostKey(t)
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: "test",
				SSHHostKey:  hostKey,
			})
			cfg, err := r.clientConfig()
			require.NoError(t, err)
			assert.NotNil(t, cfg.HostKeyCallback)
		})
	}
}
