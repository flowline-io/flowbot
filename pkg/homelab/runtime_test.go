package homelab

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func generateTestPrivateKey(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
}

func TestDockerComposeRuntime_ValidatePath(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			err := r.validatePath(tt.app)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDockerComposeRuntime_ComposeEnv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		socket    string
		wantMatch string
		wantHost  bool
	}{
		{
			name:      "DOCKER_HOST included when socket set",
			socket:    "unix:///var/run/docker.sock",
			wantMatch: "DOCKER_HOST=unix:///var/run/docker.sock",
			wantHost:  true,
		},
		{
			name:      "no DOCKER_HOST when socket is empty string",
			socket:    "",
			wantMatch: "DOCKER_HOST=",
			wantHost:  false,
		},
		{
			name:      "DOCKER_HOST included with custom socket path",
			socket:    "unix:///custom/path/docker.sock",
			wantMatch: "DOCKER_HOST=unix:///custom/path/docker.sock",
			wantHost:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket, DockerSocket: tt.socket}, "/apps")
			env := r.composeEnv()
			found := false
			for _, e := range env {
				if e == tt.wantMatch {
					found = true
					break
				}
			}
			assert.Equal(t, tt.wantHost, found)
		})
	}
}

func TestDockerComposeRuntime_ComposeEnvNoSocket(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		appsDir string
	}{
		{name: "no DOCKER_HOST when socket is empty"},
		{name: "no DOCKER_HOST when apps_dir is root", appsDir: "/"},
		{name: "no DOCKER_HOST when apps_dir is relative", appsDir: "./apps"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket}, tt.appsDir)
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
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(t *testing.T) (context.Context, App, *DockerComposeRuntime)
		wantErr bool
	}{
		{
			name: "status call without docker daemon",
			setup: func(t *testing.T) (context.Context, App, *DockerComposeRuntime) {
				tmpDir := t.TempDir()
				appDir := filepath.Join(tmpDir, "testapp")
				require.NoError(t, os.MkdirAll(appDir, 0700))
				r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket}, tmpDir)
				app := App{Name: "testapp", Path: appDir, ComposeFile: "docker-compose.yaml"}
				return t.Context(), app, r
			},
			wantErr: true,
		},
		{
			name: "status call with cancelled context",
			setup: func(t *testing.T) (context.Context, App, *DockerComposeRuntime) {
				tmpDir := t.TempDir()
				appDir := filepath.Join(tmpDir, "testapp")
				require.NoError(t, os.MkdirAll(appDir, 0700))
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket}, tmpDir)
				app := App{Name: "testapp", Path: appDir, ComposeFile: "docker-compose.yaml"}
				return ctx, app, r
			},
			wantErr: true,
		},
		{
			name: "status call with app outside apps_dir",
			setup: func(t *testing.T) (context.Context, App, *DockerComposeRuntime) {
				tmpDir := t.TempDir()
				outsideDir := filepath.Join(tmpDir, "outside")
				require.NoError(t, os.MkdirAll(outsideDir, 0700))
				r := NewDockerComposeRuntime(RuntimeConfig{Mode: RuntimeModeDockerSocket}, tmpDir)
				app := App{Name: "outsideapp", Path: outsideDir, ComposeFile: "docker-compose.yaml"}
				return t.Context(), app, r
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, app, r := tt.setup(t)
			_, err := r.Status(ctx, app)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNoopRuntime_AllOperations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		app        App
		cancelCtx  bool
		wantStatus AppStatus
		wantLogErr bool
		wantOpErrs bool
	}{
		{
			name:       "noop runtime operations",
			app:        App{Name: "testapp", Path: "/fake/path", Status: AppStatusUnknown},
			wantStatus: AppStatusUnknown,
			wantLogErr: true,
			wantOpErrs: true,
		},
		{
			name:       "noop runtime with cancelled context",
			app:        App{Name: "testapp", Path: "/fake/path", Status: AppStatusRunning},
			cancelCtx:  true,
			wantStatus: AppStatusUnknown,
			wantLogErr: true,
			wantOpErrs: true,
		},
		{
			name:       "noop runtime Status preserves app status",
			app:        App{Name: "preserve", Path: "/fake/path", Status: AppStatusStopped},
			wantStatus: AppStatusStopped,
			wantLogErr: true,
			wantOpErrs: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NoopRuntime{}
			ctx := t.Context()
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(t.Context())
				cancel()
			}

			status, err := r.Status(ctx, tt.app)
			if tt.wantStatus == AppStatusUnknown && tt.cancelCtx {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantStatus, status)

			_, err = r.Logs(ctx, tt.app, 100)
			if tt.wantLogErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.wantOpErrs {
				require.Error(t, r.Start(ctx, tt.app))
				require.Error(t, r.Stop(ctx, tt.app))
				require.Error(t, r.Restart(ctx, tt.app))
				require.Error(t, r.Pull(ctx, tt.app))
				assert.Error(t, r.Update(ctx, tt.app))
			} else {
				require.NoError(t, r.Start(ctx, tt.app))
				require.NoError(t, r.Stop(ctx, tt.app))
				require.NoError(t, r.Restart(ctx, tt.app))
				require.NoError(t, r.Pull(ctx, tt.app))
				assert.NoError(t, r.Update(ctx, tt.app))
			}
		})
	}
}

func TestNewRuntime_ReturnsCorrectType(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name     string
		sshHost  string
		sshPort  int
		sshUser  string
		wantPort int
	}{
		{
			name:     "SSH runtime config defaults",
			sshHost:  "example.com",
			sshUser:  "root",
			wantPort: 22,
		},
		{
			name:     "SSH runtime with explicit non-default port",
			sshHost:  "example.com",
			sshPort:  2222,
			sshUser:  "deployer",
			wantPort: 2222,
		},
		{
			name:     "SSH runtime with empty host preserves empty host",
			sshHost:  "",
			sshUser:  "root",
			wantPort: 22,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     tt.sshHost,
				SSHPort:     tt.sshPort,
				SSHUser:     tt.sshUser,
				SSHPassword: "test",
				SSHHostKey:  generateTestHostKey(t),
			})
			assert.Equal(t, tt.sshHost, r.host)
			assert.Equal(t, tt.wantPort, r.port)
			assert.Equal(t, tt.sshUser, r.user)
		})
	}
}

func TestSSHRuntime_ClientConfigNoAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		sshHost string
		sshPort int
		sshUser string
		wantErr string
	}{
		{
			name:    "client config fails without auth",
			sshHost: "example.com",
			sshUser: "root",
			wantErr: "key or password",
		},
		{
			name:    "client config fails without auth on different host",
			sshHost: "another.example.com",
			sshUser: "admin",
			wantErr: "key or password",
		},
		{
			name:    "client config fails without auth even with port configured",
			sshHost: "example.com",
			sshPort: 2222,
			sshUser: "root",
			wantErr: "key or password",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewSSHRuntime(RuntimeConfig{
				Mode:       RuntimeModeSSH,
				SSHHost:    tt.sshHost,
				SSHPort:    tt.sshPort,
				SSHUser:    tt.sshUser,
				SSHHostKey: generateTestHostKey(t),
			})
			_, err := r.clientConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSSHRuntime_ClientConfigPasswordAuth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		password  string
		key       string
		wantUser  string
		wantAuthN int
		wantErr   bool
	}{
		{
			name:      "client config with password auth",
			password:  "test",
			wantUser:  "root",
			wantAuthN: 1,
		},
		{
			name:      "client config with key auth prefers key over password",
			password:  "test",
			key:       "",
			wantUser:  "root",
			wantAuthN: 1,
		},
		{
			name:      "client config with private key auth",
			password:  "",
			key:       "",
			wantUser:  "root",
			wantAuthN: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hostKey := generateTestHostKey(t)
			key := tt.key
			if tt.name == "client config with private key auth" {
				key = generateTestPrivateKey(t)
			}
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     tt.wantUser,
				SSHPassword: tt.password,
				SSHKey:      key,
				SSHHostKey:  hostKey,
			})
			cfg, err := r.clientConfig()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantUser, cfg.User)
			require.Len(t, cfg.Auth, tt.wantAuthN)
		})
	}
}

func TestSSHRuntime_DefaultPort(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		sshPort  int
		wantPort int
	}{
		{name: "zero port defaults to 22", sshPort: 0, wantPort: 22},
		{name: "custom port is used", sshPort: 2222, wantPort: 2222},
		{name: "explicitly specified standard port 22", sshPort: 22, wantPort: 22},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T) context.Context
	}{
		{
			name: "cancelled context causes errors for all operations",
			setup: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
		},
		{
			name: "deadline-exceeded context causes errors for all operations",
			setup: func(t *testing.T) context.Context {
				ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Hour))
				cancel()
				return ctx
			},
		},
		{
			name: "context with zero timeout causes errors for all operations",
			setup: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(t.Context(), 0)
				time.Sleep(time.Millisecond)
				cancel()
				return ctx
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.setup(t)

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
	t.Parallel()
	tests := []struct {
		name     string
		password string
		key      string
		wantErr  string
	}{
		{
			name:     "client config fails without host key",
			password: "test",
			wantErr:  "host_key",
		},
		{
			name:    "client config fails without host key with key auth",
			key:     "",
			wantErr: "host_key",
		},
		{
			name:    "client config fails without host key with empty auth",
			wantErr: "host_key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			key := tt.key
			if tt.name == "client config fails without host key with key auth" {
				key = generateTestPrivateKey(t)
			}
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: tt.password,
				SSHKey:      key,
			})
			_, err := r.clientConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSSHRuntime_ClientConfigInvalidHostKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		hostKey string
		wantErr string
	}{
		{
			name:    "client config fails with invalid host key",
			hostKey: "not-a-valid-key",
			wantErr: "parse ssh host key",
		},
		{
			name:    "client config fails with truncated base64 host key",
			hostKey: "ssh-rsa AAAAB3NzaC",
			wantErr: "parse ssh host key",
		},
		{
			name:    "client config fails with raw binary as host key",
			hostKey: "\x00\x01\x02\xFF\xFE",
			wantErr: "parse ssh host key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     "root",
				SSHPassword: "test",
				SSHHostKey:  tt.hostKey,
			})
			_, err := r.clientConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSSHRuntime_ClientConfigFixedHostKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		password string
		key      string
		wantUser string
	}{
		{
			name:     "fixed host key sets HostKeyCallback",
			password: "test",
			wantUser: "root",
		},
		{
			name:     "HostKeyCallback is set with key auth",
			key:      "",
			wantUser: "root",
		},
		{
			name:     "HostKeyCallback is set and User is preserved",
			password: "test",
			wantUser: "deploy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hostKey := generateTestHostKey(t)
			key := tt.key
			if tt.name == "HostKeyCallback is set with key auth" {
				key = generateTestPrivateKey(t)
			}
			r := NewSSHRuntime(RuntimeConfig{
				Mode:        RuntimeModeSSH,
				SSHHost:     "example.com",
				SSHUser:     tt.wantUser,
				SSHPassword: tt.password,
				SSHKey:      key,
				SSHHostKey:  hostKey,
			})
			cfg, err := r.clientConfig()
			require.NoError(t, err)
			assert.NotNil(t, cfg.HostKeyCallback)
			assert.Equal(t, tt.wantUser, cfg.User)
		})
	}
}
