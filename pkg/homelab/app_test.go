package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppStatusConstants(t *testing.T) {
	assert.Equal(t, AppStatus("unknown"), AppStatusUnknown)
	assert.Equal(t, AppStatus("running"), AppStatusRunning)
	assert.Equal(t, AppStatus("stopped"), AppStatusStopped)
	assert.Equal(t, AppStatus("partial"), AppStatusPartial)
}

func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, HealthStatus("unknown"), HealthUnknown)
	assert.Equal(t, HealthStatus("healthy"), HealthHealthy)
	assert.Equal(t, HealthStatus("unhealthy"), HealthUnhealthy)
}

func TestRuntimeModeConstants(t *testing.T) {
	assert.Equal(t, RuntimeMode("none"), RuntimeModeNone)
	assert.Equal(t, RuntimeMode("docker_socket"), RuntimeModeDockerSocket)
	assert.Equal(t, RuntimeMode("ssh"), RuntimeModeSSH)
}

func TestAppZeroValue(t *testing.T) {
	app := App{}
	assert.Equal(t, AppStatus(""), app.Status)
	assert.Equal(t, HealthStatus(""), app.Health)
	assert.Nil(t, app.Services)
	assert.Nil(t, app.Networks)
	assert.Nil(t, app.Ports)
	assert.Nil(t, app.Labels)
}

func TestComposeServiceZeroValue(t *testing.T) {
	svc := ComposeService{}
	assert.Empty(t, svc.Name)
	assert.Empty(t, svc.Image)
	assert.Empty(t, svc.Container)
}

func TestPortMappingZeroValue(t *testing.T) {
	pm := PortMapping{}
	assert.Empty(t, pm.Host)
	assert.Empty(t, pm.HostPort)
	assert.Empty(t, pm.Container)
	assert.Empty(t, pm.Protocol)
}

func TestPermissionsZeroValue(t *testing.T) {
	p := Permissions{}
	assert.False(t, p.Status)
	assert.False(t, p.Logs)
	assert.False(t, p.Start)
	assert.False(t, p.Stop)
	assert.False(t, p.Restart)
	assert.False(t, p.Pull)
	assert.False(t, p.Update)
	assert.False(t, p.Exec)
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{}
	assert.Empty(t, cfg.Root)
	assert.Empty(t, cfg.AppsDir)
	assert.Empty(t, cfg.ComposeFile)
	assert.Nil(t, cfg.Allowlist)
	assert.Equal(t, RuntimeMode(""), cfg.Runtime.Mode)
}

func TestRuntimeConfigDefaults(t *testing.T) {
	rc := RuntimeConfig{}
	assert.Equal(t, RuntimeMode(""), rc.Mode)
	assert.Empty(t, rc.DockerSocket)
	assert.Equal(t, 0, rc.SSHPort)
}
