package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/homelab/probe"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestHomelabConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  config.Homelab
		want homelab.Config
	}{
		{
			name: "empty config returns defaults for timeout/concurrency",
			cfg:  config.Homelab{},
			want: homelab.Config{
				Discovery: homelab.DiscoveryConfig{
					ProbeTimeout:     5 * time.Second,
					ProbeConcurrency: 4,
				},
			},
		},
		{
			name: "full config propagates all fields",
			cfg: config.Homelab{
				Root:        "/opt/homelab",
				AppsDir:     "/opt/homelab/apps",
				ComposeFile: "docker-compose.prod.yml",
				Allowlist:   []string{"app-a", "app-b"},
				Runtime: config.HomelabRuntime{
					Mode:         "docker_socket",
					DockerSocket: "/var/run/docker.sock",
					SSHHost:      "192.168.1.10",
					SSHPort:      22,
					SSHUser:      "admin",
					SSHPassword:  "pass",
					SSHKey:       "key-content",
					SSHHostKey:   "hostkey",
				},
				Permissions: config.HomelabPermissions{
					Status:  true,
					Logs:    true,
					Start:   false,
					Stop:    false,
					Restart: true,
					Pull:    true,
					Update:  false,
					Exec:    true,
				},
				Discovery: config.HomelabDiscovery{
					ProbeEnabled:       true,
					ProbeTimeout:       "10s",
					ProbeConcurrency:   8,
					FingerprintEnabled: true,
					LabelPriority:      true,
				},
			},
			want: homelab.Config{
				Root:        "/opt/homelab",
				AppsDir:     "/opt/homelab/apps",
				ComposeFile: "docker-compose.prod.yml",
				Allowlist:   []string{"app-a", "app-b"},
				Runtime: homelab.RuntimeConfig{
					Mode:         homelab.RuntimeModeDockerSocket,
					DockerSocket: "/var/run/docker.sock",
					SSHHost:      "192.168.1.10",
					SSHPort:      22,
					SSHUser:      "admin",
					SSHPassword:  "pass",
					SSHKey:       "key-content",
					SSHHostKey:   "hostkey",
				},
				Permissions: homelab.Permissions{
					Status:  true,
					Logs:    true,
					Start:   false,
					Stop:    false,
					Restart: true,
					Pull:    true,
					Update:  false,
					Exec:    true,
				},
				Discovery: homelab.DiscoveryConfig{
					ProbeEnabled:       true,
					ProbeTimeout:       10 * time.Second,
					ProbeConcurrency:   8,
					FingerprintEnabled: true,
					LabelPriority:      true,
				},
			},
		},
		{
			name: "invalid timeout falls back to default 5s",
			cfg: config.Homelab{
				Discovery: config.HomelabDiscovery{
					ProbeTimeout: "invalid",
				},
			},
			want: homelab.Config{
				Discovery: homelab.DiscoveryConfig{
					ProbeTimeout:     5 * time.Second,
					ProbeConcurrency: 4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := homelabConfig(tt.cfg)
			assert.Equal(t, tt.want.Root, got.Root)
			assert.Equal(t, tt.want.AppsDir, got.AppsDir)
			assert.Equal(t, tt.want.ComposeFile, got.ComposeFile)
			assert.Equal(t, tt.want.Allowlist, got.Allowlist)
			assert.Equal(t, tt.want.Runtime, got.Runtime)
			assert.Equal(t, tt.want.Permissions, got.Permissions)
			assert.Equal(t, tt.want.Discovery, got.Discovery)
		})
	}
}

func TestMergeProbeResults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		apps         []homelab.App
		probeResults []probe.ProbeResult
		want         []homelab.App
	}{
		{
			name:         "empty apps returns empty",
			apps:         nil,
			probeResults: nil,
			want:         nil,
		},
		{
			name: "app without capabilities gets probe capabilities",
			apps: []homelab.App{
				{
					Name:         "app1",
					Capabilities: nil,
				},
			},
			probeResults: []probe.ProbeResult{
				{
					AppName: "app1",
					Capabilities: []homelab.AppCapability{
						{Capability: homelab.CapKarakeep},
					},
				},
			},
			want: []homelab.App{
				{
					Name: "app1",
					Capabilities: []homelab.AppCapability{
						{Capability: homelab.CapKarakeep},
					},
				},
			},
		},
		{
			name: "app with existing capabilities enriches with probe endpoint/auth",
			apps: []homelab.App{
				{
					Name: "app2",
					Capabilities: []homelab.AppCapability{
						{Capability: homelab.CapMiniflux, Endpoint: nil, Auth: nil},
					},
				},
			},
			probeResults: []probe.ProbeResult{
				{
					AppName: "app2",
					Capabilities: []homelab.AppCapability{
						{
							Capability: homelab.CapMiniflux,
							Endpoint:   &homelab.EndpointInfo{BaseURL: "http://miniflux:8080", Health: "/health"},
							Auth:       &homelab.AuthInfo{Type: homelab.AuthAPIToken, Header: "X-Auth-Token"},
						},
					},
				},
			},
			want: []homelab.App{
				{
					Name: "app2",
					Capabilities: []homelab.AppCapability{
						{
							Capability: homelab.CapMiniflux,
							Endpoint:   &homelab.EndpointInfo{BaseURL: "http://miniflux:8080", Health: "/health"},
							Auth:       &homelab.AuthInfo{Type: homelab.AuthAPIToken, Header: "X-Auth-Token"},
						},
					},
				},
			},
		},
		{
			name: "app with existing endpoint is not overwritten",
			apps: []homelab.App{
				{
					Name: "app3",
					Capabilities: []homelab.AppCapability{
						{
							Capability: homelab.CapArchive,
							Endpoint:   &homelab.EndpointInfo{BaseURL: "http://custom:9000", Health: "/ready"},
							Auth:       nil,
						},
					},
				},
			},
			probeResults: []probe.ProbeResult{
				{
					AppName: "app3",
					Capabilities: []homelab.AppCapability{
						{
							Capability: homelab.CapArchive,
							Endpoint:   &homelab.EndpointInfo{BaseURL: "http://probed:9000", Health: "/probe"},
							Auth:       &homelab.AuthInfo{Type: homelab.AuthBasic},
						},
					},
				},
			},
			want: []homelab.App{
				{
					Name: "app3",
					Capabilities: []homelab.AppCapability{
						{
							Capability: homelab.CapArchive,
							Endpoint:   &homelab.EndpointInfo{BaseURL: "http://custom:9000", Health: "/ready"},
							Auth:       &homelab.AuthInfo{Type: homelab.AuthBasic},
						},
					},
				},
			},
		},
		{
			name: "app not in probe results is unchanged",
			apps: []homelab.App{
				{
					Name:         "untouched",
					Capabilities: nil,
				},
			},
			probeResults: []probe.ProbeResult{
				{
					AppName: "other-app",
					Capabilities: []homelab.AppCapability{
						{Capability: homelab.CapKanboard},
					},
				},
			},
			want: []homelab.App{
				{
					Name:         "untouched",
					Capabilities: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mergeProbeResults(tt.apps, tt.probeResults)
			assert.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.Equal(t, tt.want[i].Name, got[i].Name)
				assert.Equal(t, tt.want[i].Capabilities, got[i].Capabilities)
			}
		})
	}
}

func TestRunHomelabScan(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  config.Homelab
	}{
		{
			name: "disabled_config_returns_error",
			cfg:  config.Homelab{AppsDir: "", Root: ""},
		},
		{
			name: "nonexistent_apps_dir_returns_error",
			cfg:  config.Homelab{AppsDir: "/nonexistent/path", Root: ""},
		},
		{
			name: "nonexistent_root_returns_error",
			cfg:  config.Homelab{AppsDir: "", Root: "/nonexistent/root"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := RunHomelabScan(tt.cfg)
			assert.Error(t, err)
		})
	}
}

func TestDataEventFromImageUpdate(t *testing.T) {
	t.Parallel()
	ev := dataEventFromImageUpdate(homelab.ImageUpdate{
		AppName:       "karakeep",
		Service:       "web",
		Image:         "ghcr.io/karakeep/karakeep:latest",
		Tag:           "latest",
		CurrentDigest: "sha256:old",
		RemoteDigest:  "sha256:new",
		Capability:    "karakeep",
	})
	assert.Equal(t, types.EventHomelabImageUpdateAvailable, ev.EventType)
	assert.Equal(t, homelab.ImageCheckSource, ev.Source)
	assert.Equal(t, "karakeep", ev.App)
	assert.Equal(t, "karakeep", ev.Capability)
	assert.Equal(t, "karakeep/web", ev.EntityID)
	assert.Equal(t, "karakeep/web/sha256:new", ev.IdempotencyKey)
	assert.NotEmpty(t, ev.EventID)
	assert.Equal(t, "ghcr.io/karakeep/karakeep:latest", ev.Data["image"])
	assert.Equal(t, "latest", ev.Data["tag"])
	assert.Equal(t, "sha256:old", ev.Data["current_digest"])
	assert.Equal(t, "sha256:new", ev.Data["remote_digest"])
}

func TestPublishHomelabImageUpdates(t *testing.T) {
	t.Parallel()
	u := homelab.ImageUpdate{AppName: "app", Service: "web", RemoteDigest: "sha256:new"}
	tests := []struct {
		name      string
		exists    func(context.Context, string, string) (bool, error)
		emitErr   error
		wantEmits int
	}{
		{
			name:      "emits when not seen",
			exists:    func(context.Context, string, string) (bool, error) { return false, nil },
			wantEmits: 1,
		},
		{
			name:      "skips when already recorded",
			exists:    func(context.Context, string, string) (bool, error) { return true, nil },
			wantEmits: 0,
		},
		{
			name:      "skips emit when exists lookup fails",
			exists:    func(context.Context, string, string) (bool, error) { return false, errors.New("db") },
			wantEmits: 0,
		},
		{
			name:      "records the emit attempt when persist fails",
			exists:    func(context.Context, string, string) (bool, error) { return false, nil },
			emitErr:   errors.New("publish"),
			wantEmits: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			emits := 0
			publishHomelabImageUpdates(t.Context(), []homelab.ImageUpdate{u}, tt.exists, func(_ context.Context, de types.DataEvent) error {
				emits++
				require.Equal(t, types.EventHomelabImageUpdateAvailable, de.EventType)
				return tt.emitErr
			})
			assert.Equal(t, tt.wantEmits, emits)
		})
	}
}
