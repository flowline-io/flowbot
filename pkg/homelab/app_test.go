package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppStatusConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected AppStatus
	}{
		{name: "unknown", input: "unknown", expected: AppStatusUnknown},
		{name: "running", input: "running", expected: AppStatusRunning},
		{name: "stopped", input: "stopped", expected: AppStatusStopped},
		{name: "partial", input: "partial", expected: AppStatusPartial},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, AppStatus(tt.input))
		})
	}
}

func TestHealthStatusConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected HealthStatus
	}{
		{name: "unknown", input: "unknown", expected: HealthUnknown},
		{name: "healthy", input: "healthy", expected: HealthHealthy},
		{name: "unhealthy", input: "unhealthy", expected: HealthUnhealthy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, HealthStatus(tt.input))
		})
	}
}

func TestRuntimeModeConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected RuntimeMode
	}{
		{name: "none", input: "none", expected: RuntimeModeNone},
		{name: "docker_socket", input: "docker_socket", expected: RuntimeModeDockerSocket},
		{name: "ssh", input: "ssh", expected: RuntimeModeSSH},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, RuntimeMode(tt.input))
		})
	}
}

func TestAppZeroValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		app  App
	}{
		{name: "zero value app has empty/default fields", app: App{}},
		{name: "app with name only, other fields zero", app: App{Name: "test-app"}},
		{name: "app with empty capabilities slice", app: App{Capabilities: []AppCapability{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, AppStatus(""), tt.app.Status)
			assert.Equal(t, HealthStatus(""), tt.app.Health)
			assert.Nil(t, tt.app.Services)
			assert.Nil(t, tt.app.Networks)
			assert.Nil(t, tt.app.Ports)
			assert.Nil(t, tt.app.Labels)
		})
	}
}

func TestComposeServiceZeroValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		svc           ComposeService
		expectedImage string
		expectedName  string
	}{
		{name: "zero value compose service", svc: ComposeService{}, expectedImage: "", expectedName: ""},
		{name: "service with name only, rest empty", svc: ComposeService{Name: "web"}, expectedImage: "", expectedName: "web"},
		{name: "service with image only, rest empty", svc: ComposeService{Image: "nginx:latest"}, expectedImage: "nginx:latest", expectedName: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expectedName, tt.svc.Name)
			assert.Equal(t, tt.expectedImage, tt.svc.Image)
			assert.Empty(t, tt.svc.Container)
		})
	}
}

func TestPortMappingZeroValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		pm             PortMapping
		expectHost     string
		expectProtocol string
	}{
		{name: "zero value port mapping", pm: PortMapping{}, expectHost: "", expectProtocol: ""},
		{name: "port mapping with host only", pm: PortMapping{Host: "0.0.0.0"}, expectHost: "0.0.0.0", expectProtocol: ""},
		{name: "port mapping with protocol only", pm: PortMapping{Protocol: "udp"}, expectHost: "", expectProtocol: "udp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expectHost, tt.pm.Host)
			assert.Equal(t, tt.expectProtocol, tt.pm.Protocol)
			assert.Empty(t, tt.pm.HostPort)
			assert.Empty(t, tt.pm.Container)
		})
	}
}

func TestPermissionsZeroValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		p           Permissions
		expectStart bool
		expectExec  bool
	}{
		{name: "zero value permissions are all false", p: Permissions{}, expectStart: false, expectExec: false},
		{name: "only start true, rest false", p: Permissions{Start: true}, expectStart: true, expectExec: false},
		{name: "only exec true, rest false", p: Permissions{Exec: true}, expectStart: false, expectExec: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.False(t, tt.p.Status)
			assert.False(t, tt.p.Logs)
			assert.Equal(t, tt.expectStart, tt.p.Start)
			assert.False(t, tt.p.Stop)
			assert.False(t, tt.p.Restart)
			assert.False(t, tt.p.Pull)
			assert.False(t, tt.p.Update)
			assert.Equal(t, tt.expectExec, tt.p.Exec)
		})
	}
}

func TestAllowsLifecycle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		perm      Permissions
		operation string
		want      bool
	}{
		{name: "start allowed", perm: Permissions{Start: true}, operation: "start", want: true},
		{name: "start denied", perm: Permissions{}, operation: "start", want: false},
		{name: "pull allowed", perm: Permissions{Pull: true}, operation: "pull", want: true},
		{name: "update denied when only start", perm: Permissions{Start: true}, operation: "update", want: false},
		{name: "unknown operation denied", perm: Permissions{Start: true}, operation: "exec", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, AllowsLifecycle(tt.perm, tt.operation))
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		cfg        Config
		expectRoot string
	}{
		{name: "zero value config", cfg: Config{}, expectRoot: ""},
		{name: "config with root set, others default", cfg: Config{Root: "/data"}, expectRoot: "/data"},
		{name: "config with empty allowlist", cfg: Config{Allowlist: []string{}}, expectRoot: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expectRoot, tt.cfg.Root)
			assert.Empty(t, tt.cfg.AppsDir)
			assert.Empty(t, tt.cfg.ComposeFile)
			assert.Equal(t, RuntimeMode(""), tt.cfg.Runtime.Mode)
		})
	}
}

func TestRuntimeConfigDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		rc           RuntimeConfig
		expectPort   int
		expectSocket string
	}{
		{name: "zero value runtime config", rc: RuntimeConfig{}, expectPort: 0, expectSocket: ""},
		{name: "runtime config with ssh port", rc: RuntimeConfig{SSHPort: 22}, expectPort: 22, expectSocket: ""},
		{name: "runtime config with docker socket", rc: RuntimeConfig{DockerSocket: "/var/run/docker.sock"}, expectPort: 0, expectSocket: "/var/run/docker.sock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, RuntimeMode(""), tt.rc.Mode)
			assert.Equal(t, tt.expectSocket, tt.rc.DockerSocket)
			assert.Equal(t, tt.expectPort, tt.rc.SSHPort)
		})
	}
}
