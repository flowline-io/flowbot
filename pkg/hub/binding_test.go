package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindings(t *testing.T) {
	tests := []struct {
		name  string
		check func(*testing.T, *Registry)
	}{
		{
			name: "empty registry returns empty bindings",
			check: func(t *testing.T, r *Registry) {
				bindings := r.Bindings()
				assert.Empty(t, bindings)
			},
		},
		{
			name: "returns sorted bindings",
			check: func(t *testing.T, r *Registry) {
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
			},
		},
		{
			name: "reflects health changes",
			check: func(t *testing.T, r *Registry) {
				require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: true}))

				bindings := r.Bindings()
				require.Len(t, bindings, 1)
				assert.True(t, bindings[0].Healthy)

				require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep", Healthy: false}))

				bindings = r.Bindings()
				require.Len(t, bindings, 1)
				assert.False(t, bindings[0].Healthy)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			tt.check(t, r)
		})
	}
}

func TestBindingJsonTags(t *testing.T) {
	t.Run("struct fields are set correctly", func(t *testing.T) {
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
	})
}
