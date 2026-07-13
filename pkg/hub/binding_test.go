package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindings(t *testing.T) {
	t.Parallel()

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
				require.NoError(t, r.Register(Descriptor{Type: CapKanboard, App: "kanboard", Healthy: true}))
				require.NoError(t, r.Register(Descriptor{Type: CapExample, App: "example", Healthy: false}))
				require.NoError(t, r.Register(Descriptor{Type: CapKarakeep, App: "karakeep", Healthy: true}))

				bindings := r.Bindings()
				require.Len(t, bindings, 3)

				assert.Equal(t, CapExample, bindings[0].Capability)
				assert.Equal(t, "example", bindings[0].App)
				assert.False(t, bindings[0].Healthy)

				assert.Equal(t, CapKanboard, bindings[1].Capability)
				assert.Equal(t, "kanboard", bindings[1].App)
				assert.True(t, bindings[1].Healthy)

				assert.Equal(t, CapKarakeep, bindings[2].Capability)
				assert.Equal(t, "karakeep", bindings[2].App)
				assert.True(t, bindings[2].Healthy)
			},
		},
		{
			name: "reflects health changes",
			check: func(t *testing.T, r *Registry) {
				require.NoError(t, r.Register(Descriptor{Type: CapKarakeep, App: "karakeep", Healthy: true}))

				bindings := r.Bindings()
				require.Len(t, bindings, 1)
				assert.True(t, bindings[0].Healthy)

				require.NoError(t, r.Register(Descriptor{Type: CapKarakeep, App: "karakeep", Healthy: false}))

				bindings = r.Bindings()
				require.Len(t, bindings, 1)
				assert.False(t, bindings[0].Healthy)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			tt.check(t, r)
		})
	}
}

func TestBindingJsonTags(t *testing.T) {
	t.Parallel()

	t.Run("struct fields are set correctly", func(t *testing.T) {
		t.Parallel()
		b := Binding{
			Capability: CapKarakeep,
			App:        "karakeep",
			Healthy:    true,
		}
		assert.Equal(t, CapKarakeep, b.Capability)
		assert.Equal(t, "karakeep", b.App)
		assert.True(t, b.Healthy)
	})
}
