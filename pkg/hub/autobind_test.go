package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestAutoBind(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T)
		check func(*testing.T, []DiscoveredBinding)
	}{
		{
			name: "empty homelab registry",
			setup: func(_ *testing.T) {
			},
			check: func(t *testing.T, bindings []DiscoveredBinding) {
				assert.Empty(t, bindings)
			},
		},
		{
			name: "discovered capabilities",
			setup: func(_ *testing.T) {
				apps := []homelab.App{
					{
						Name: "my-karakeep",
						Capabilities: []homelab.AppCapability{
							{
								Capability: homelab.CapBookmark,
								Backend:    "karakeep",
								Endpoint: &homelab.EndpointInfo{
									BaseURL: "http://localhost:3000",
									Health:  "/health",
								},
								Auth: &homelab.AuthInfo{
									Type:   homelab.AuthAPIToken,
									Header: "Authorization",
									Prefix: "Bearer",
								},
							},
						},
					},
				}
				homelab.DefaultRegistry.Replace(apps)
			},
			check: func(t *testing.T, bindings []DiscoveredBinding) {
				require.Len(t, bindings, 1)
				assert.Equal(t, CapBookmark, bindings[0].Capability)
				assert.Equal(t, "karakeep", bindings[0].Backend)
				assert.Equal(t, "my-karakeep", bindings[0].App)
				assert.False(t, bindings[0].Bound)
				require.NotNil(t, bindings[0].Endpoint)
				assert.Equal(t, "http://localhost:3000", bindings[0].Endpoint.BaseURL)
				assert.Equal(t, "/health", bindings[0].Endpoint.Health)
				require.NotNil(t, bindings[0].Auth)
				assert.Equal(t, "api_token", string(bindings[0].Auth.Type))
			},
		},
		{
			name: "already bound capability",
			setup: func(t *testing.T) {
				err := Default.Register(Descriptor{
					Type:    CapBookmark,
					Backend: "karakeep",
					App:     "my-karakeep",
				})
				require.NoError(t, err)

				apps := []homelab.App{
					{
						Name: "my-karakeep",
						Capabilities: []homelab.AppCapability{
							{Capability: homelab.CapBookmark, Backend: "karakeep"},
						},
					},
				}
				homelab.DefaultRegistry.Replace(apps)
			},
			check: func(t *testing.T, bindings []DiscoveredBinding) {
				require.Len(t, bindings, 1)
				assert.True(t, bindings[0].Bound)
			},
		},
		{
			name: "multiple apps with capabilities",
			setup: func(_ *testing.T) {
				apps := []homelab.App{
					{
						Name: "archive",
						Capabilities: []homelab.AppCapability{
							{Capability: homelab.CapArchive, Backend: "archivebox"},
						},
					},
					{
						Name: "rss",
						Capabilities: []homelab.AppCapability{
							{Capability: homelab.CapReader, Backend: "miniflux"},
						},
					},
				}
				homelab.DefaultRegistry.Replace(apps)
			},
			check: func(t *testing.T, bindings []DiscoveredBinding) {
				require.Len(t, bindings, 2)

				capMap := make(map[string]string)
				for _, b := range bindings {
					capMap[string(b.Capability)] = b.App
				}
				assert.Equal(t, "archive", capMap["archive"])
				assert.Equal(t, "rss", capMap["reader"])
			},
		},
		{
			name: "app without capabilities",
			setup: func(_ *testing.T) {
				apps := []homelab.App{
					{Name: "no-labels", Capabilities: nil},
					{
						Name: "has-labels",
						Capabilities: []homelab.AppCapability{
							{Capability: homelab.CapKanban, Backend: "kanboard"},
						},
					},
				}
				homelab.DefaultRegistry.Replace(apps)
			},
			check: func(t *testing.T, bindings []DiscoveredBinding) {
				require.Len(t, bindings, 1)
				assert.Equal(t, "kanban", string(bindings[0].Capability))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRegistry := homelab.DefaultRegistry
			homelab.DefaultRegistry = homelab.NewRegistry()
			defer func() { homelab.DefaultRegistry = oldRegistry }()

			oldDefault := Default
			Default = NewRegistry()
			defer func() { Default = oldDefault }()

			tt.setup(t)
			bindings := AutoBind()
			tt.check(t, bindings)
		})
	}
}
