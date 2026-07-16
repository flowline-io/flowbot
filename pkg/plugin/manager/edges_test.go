package manager

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/plugin"
)

func TestDeriveIdentity(t *testing.T) {
	t.Parallel()
	manifest := &plugin.Manifest{Name: "my-plugin"}
	tests := []struct {
		name    string
		srcType string
		want    string
	}{
		{name: "local source uses manifest name", srcType: "local", want: "my-plugin"},
		{name: "oci source uses manifest name", srcType: "oci", want: "my-plugin"},
		{name: "git source uses manifest name", srcType: "git", want: "my-plugin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deriveIdentity(SourceConfig{Type: tt.srcType}, manifest)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringSlicesEqual(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		a     []string
		b     []string
		equal bool
	}{
		{name: "both nil", a: nil, b: nil, equal: true},
		{name: "same elements", a: []string{"a", "b"}, b: []string{"a", "b"}, equal: true},
		{name: "different length", a: []string{"a"}, b: []string{"a", "b"}, equal: false},
		{name: "different values", a: []string{"a"}, b: []string{"b"}, equal: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.equal, stringSlicesEqual(tt.a, tt.b))
		})
	}
}

func TestInitWithEmptyLocalSource(t *testing.T) {
	t.Parallel()
	mgr := NewPluginManager(&PluginConfig{
		Enabled: true,
		Sources: []SourceConfig{
			{Type: "local", Path: t.TempDir()},
		},
	}, zerolog.Nop())
	err := mgr.Init(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, mgr.List())
}

func TestInitInvalidSourceType(t *testing.T) {
	t.Parallel()
	mgr := NewPluginManager(&PluginConfig{
		Enabled: true,
		Sources: []SourceConfig{
			{Type: "unknown"},
		},
	}, zerolog.Nop())
	err := mgr.Init(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, mgr.List())
}
