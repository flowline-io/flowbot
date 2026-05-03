package hub

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoBind_EmptyRegistry(t *testing.T) {
	// Reset homelab registry to a clean state.
	oldRegistry := homelab.DefaultRegistry
	homelab.DefaultRegistry = homelab.NewRegistry()
	defer func() { homelab.DefaultRegistry = oldRegistry }()

	bindings := AutoBind()
	assert.Empty(t, bindings)
}

func TestAutoBind_DiscoveredCapabilities(t *testing.T) {
	oldRegistry := homelab.DefaultRegistry
	homelab.DefaultRegistry = homelab.NewRegistry()
	defer func() { homelab.DefaultRegistry = oldRegistry }()

	// Register a homelab app with discovered capabilities.
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

	bindings := AutoBind()
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
}

func TestAutoBind_AlreadyBoundCapability(t *testing.T) {
	oldRegistry := homelab.DefaultRegistry
	homelab.DefaultRegistry = homelab.NewRegistry()
	defer func() { homelab.DefaultRegistry = oldRegistry }()

	// Register a full descriptor in the hub.
	err := Default.Register(Descriptor{
		Type:    CapBookmark,
		Backend: "karakeep",
		App:     "my-karakeep",
	})
	require.NoError(t, err)

	// Register matching app in homelab.
	apps := []homelab.App{
		{
			Name: "my-karakeep",
			Capabilities: []homelab.AppCapability{
				{Capability: homelab.CapBookmark, Backend: "karakeep"},
			},
		},
	}
	homelab.DefaultRegistry.Replace(apps)

	bindings := AutoBind()
	require.Len(t, bindings, 1)
	assert.True(t, bindings[0].Bound)
}

func TestAutoBind_MultipleAppsWithCapabilities(t *testing.T) {
	oldRegistry := homelab.DefaultRegistry
	homelab.DefaultRegistry = homelab.NewRegistry()
	defer func() { homelab.DefaultRegistry = oldRegistry }()

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

	bindings := AutoBind()
	require.Len(t, bindings, 2)

	capMap := make(map[string]string)
	for _, b := range bindings {
		capMap[string(b.Capability)] = b.App
	}
	assert.Equal(t, "archive", capMap["archive"])
	assert.Equal(t, "rss", capMap["reader"])
}

func TestAutoBind_AppWithoutCapabilities(t *testing.T) {
	oldRegistry := homelab.DefaultRegistry
	homelab.DefaultRegistry = homelab.NewRegistry()
	defer func() { homelab.DefaultRegistry = oldRegistry }()

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

	bindings := AutoBind()
	require.Len(t, bindings, 1)
	assert.Equal(t, "kanban", string(bindings[0].Capability))
}
