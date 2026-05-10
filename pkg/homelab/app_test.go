package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppStatusConstants(t *testing.T) {
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
			assert.Equal(t, tt.expected, AppStatus(tt.input))
		})
	}
}

func TestHealthStatusConstants(t *testing.T) {
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
			assert.Equal(t, tt.expected, HealthStatus(tt.input))
		})
	}
}

func TestRuntimeModeConstants(t *testing.T) {
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
			assert.Equal(t, tt.expected, RuntimeMode(tt.input))
		})
	}
}

func TestAppZeroValue(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "zero value app has empty/default fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := App{}
			assert.Equal(t, AppStatus(""), app.Status)
			assert.Equal(t, HealthStatus(""), app.Health)
			assert.Nil(t, app.Services)
			assert.Nil(t, app.Networks)
			assert.Nil(t, app.Ports)
			assert.Nil(t, app.Labels)
		})
	}
}

func TestComposeServiceZeroValue(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "zero value compose service"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := ComposeService{}
			assert.Empty(t, svc.Name)
			assert.Empty(t, svc.Image)
			assert.Empty(t, svc.Container)
		})
	}
}

func TestPortMappingZeroValue(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "zero value port mapping"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := PortMapping{}
			assert.Empty(t, pm.Host)
			assert.Empty(t, pm.HostPort)
			assert.Empty(t, pm.Container)
			assert.Empty(t, pm.Protocol)
		})
	}
}

func TestPermissionsZeroValue(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "zero value permissions are all false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Permissions{}
			assert.False(t, p.Status)
			assert.False(t, p.Logs)
			assert.False(t, p.Start)
			assert.False(t, p.Stop)
			assert.False(t, p.Restart)
			assert.False(t, p.Pull)
			assert.False(t, p.Update)
			assert.False(t, p.Exec)
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "zero value config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{}
			assert.Empty(t, cfg.Root)
			assert.Empty(t, cfg.AppsDir)
			assert.Empty(t, cfg.ComposeFile)
			assert.Nil(t, cfg.Allowlist)
			assert.Equal(t, RuntimeMode(""), cfg.Runtime.Mode)
		})
	}
}

func TestRuntimeConfigDefaults(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "zero value runtime config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := RuntimeConfig{}
			assert.Equal(t, RuntimeMode(""), rc.Mode)
			assert.Empty(t, rc.DockerSocket)
			assert.Equal(t, 0, rc.SSHPort)
		})
	}
}
