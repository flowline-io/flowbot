package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindingsEmpty(t *testing.T) {
	r := NewRegistry()
	bindings := r.Bindings()
	assert.Empty(t, bindings)
}

func TestBindingsReturnsSorted(t *testing.T) {
	r := NewRegistry()

	require.NoError(t, r.Register(Descriptor{Type: CapKanban, Backend: "kanboard", App: "kanboard", Healthy: true}))
	require.NoError(t, r.Register(Descriptor{Type: CapArchive, Backend: "archivebox", App: "archivebox", Healthy: false}))
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true}))

	bindings := r.Bindings()
	require.Len(t, bindings, 3)

	assert.Equal(t, CapArchive, bindings[0].Capability)
	assert.Equal(t, "archivebox", bindings[0].Backend)
	assert.Equal(t, "archivebox", bindings[0].App)
	assert.False(t, bindings[0].Healthy)

	assert.Equal(t, CapBookmark, bindings[1].Capability)
	assert.Equal(t, "karakeep", bindings[1].Backend)
	assert.True(t, bindings[1].Healthy)

	assert.Equal(t, CapKanban, bindings[2].Capability)
}

func TestBindingsReflectsHealth(t *testing.T) {
	r := NewRegistry()

	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true}))

	bindings := r.Bindings()
	require.Len(t, bindings, 1)
	assert.True(t, bindings[0].Healthy)

	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: false}))

	bindings = r.Bindings()
	require.Len(t, bindings, 1)
	assert.False(t, bindings[0].Healthy)
}

func TestBindingJsonTags(t *testing.T) {
	b := Binding{
		Capability: CapBookmark,
		Backend:    "karakeep",
		App:        "karakeep",
		Healthy:    true,
	}
	assert.Equal(t, CapBookmark, b.Capability)
	assert.Equal(t, "karakeep", b.Backend)
	assert.Equal(t, "karakeep", b.App)
	assert.True(t, b.Healthy)
}
