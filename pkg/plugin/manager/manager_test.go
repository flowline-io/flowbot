package manager

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/plugin"
)

func TestPluginManagerDisabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *PluginConfig
	}{
		{
			name:   "disabled manager does nothing",
			config: &PluginConfig{Enabled: false},
		},
		{
			name:   "nil config is no-op",
			config: nil,
		},
		{
			name: "enabled manager processes empty sources",
			config: &PluginConfig{
				Enabled: true,
				Sources: []SourceConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewPluginManager(tt.config, zerolog.Nop())
			assert.NotNil(t, mgr)

			err := mgr.Init(context.Background(), nil)
			assert.NoError(t, err)
		})
	}
}

func TestListEmpty(t *testing.T) {
	t.Parallel()

	mgr := NewPluginManager(DefaultPluginConfig(), zerolog.Nop())
	list := mgr.List()
	assert.Empty(t, list)
}

func TestUnloadNotFound(t *testing.T) {
	t.Parallel()

	mgr := NewPluginManager(DefaultPluginConfig(), zerolog.Nop())
	err := mgr.UnloadPlugin(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReloadNotFound(t *testing.T) {
	t.Parallel()

	mgr := NewPluginManager(DefaultPluginConfig(), zerolog.Nop())
	err := mgr.ReloadPlugin(context.Background(), "nonexistent", &plugin.Manifest{Name: "test"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProvidesEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		a     plugin.Provides
		b     plugin.Provides
		equal bool
	}{
		{
			name:  "identical empty",
			a:     plugin.Provides{},
			b:     plugin.Provides{},
			equal: true,
		},
		{
			name:  "identical module only",
			a:     plugin.Provides{Module: true},
			b:     plugin.Provides{Module: true},
			equal: true,
		},
		{
			name:  "module mismatch",
			a:     plugin.Provides{Module: true},
			b:     plugin.Provides{Module: false},
			equal: false,
		},
		{
			name:  "abilities match",
			a:     plugin.Provides{Abilities: []plugin.AbilityDecl{{Capability: "karakeep", Operations: []string{"list"}}}},
			b:     plugin.Provides{Abilities: []plugin.AbilityDecl{{Capability: "karakeep", Operations: []string{"list"}}}},
			equal: true,
		},
		{
			name:  "abilities capability mismatch",
			a:     plugin.Provides{Abilities: []plugin.AbilityDecl{{Capability: "karakeep", Operations: []string{"list"}}}},
			b:     plugin.Provides{Abilities: []plugin.AbilityDecl{{Capability: "notes", Operations: []string{"list"}}}},
			equal: false,
		},
		{
			name:  "abilities operations mismatch",
			a:     plugin.Provides{Abilities: []plugin.AbilityDecl{{Capability: "karakeep", Operations: []string{"list"}}}},
			b:     plugin.Provides{Abilities: []plugin.AbilityDecl{{Capability: "karakeep", Operations: []string{"list", "get"}}}},
			equal: false,
		},
		{
			name:  "provider mismatch nil vs non-nil",
			a:     plugin.Provides{Provider: nil},
			b:     plugin.Provides{Provider: &plugin.ProviderDecl{Name: "test"}},
			equal: false,
		},
		{
			name:  "provider match with oauth",
			a:     plugin.Provides{Provider: &plugin.ProviderDecl{Name: "test", OAuth: true}},
			b:     plugin.Provides{Provider: &plugin.ProviderDecl{Name: "test", OAuth: true}},
			equal: true,
		},
		{
			name:  "provider oauth mismatch",
			a:     plugin.Provides{Provider: &plugin.ProviderDecl{Name: "test", OAuth: true}},
			b:     plugin.Provides{Provider: &plugin.ProviderDecl{Name: "test", OAuth: false}},
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, providesEqual(tt.a, tt.b))
		})
	}
}
