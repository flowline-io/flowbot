package hub

import (
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)
	assert.Empty(t, r.List())
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	desc := Descriptor{
		Type:        CapBookmark,
		Backend:     "karakeep",
		App:         "karakeep",
		Description: "Bookmark service",
		Healthy:     true,
	}
	err := r.Register(desc)
	require.NoError(t, err)

	got, ok := r.Get(CapBookmark)
	assert.True(t, ok)
	assert.Equal(t, desc.Type, got.Type)
	assert.Equal(t, desc.Backend, got.Backend)
	assert.Equal(t, desc.App, got.App)
	assert.True(t, got.Healthy)
}

func TestRegistry_RegisterEmptyType(t *testing.T) {
	r := NewRegistry()

	err := r.Register(Descriptor{Backend: "test", App: "test"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, types.ErrInvalidArgument))
	assert.Contains(t, err.Error(), "capability type is required")
}

func TestRegistry_RegisterOverwrites(t *testing.T) {
	r := NewRegistry()

	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep", App: "karakeep"}))
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "linkwarden", App: "linkwarden"}))

	got, ok := r.Get(CapBookmark)
	assert.True(t, ok)
	assert.Equal(t, "linkwarden", got.Backend)
}

func TestRegistry_GetMissing(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get(CapArchive)
	assert.False(t, ok)
}

func TestRegistry_GetEmptyRegistry(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get(CapReader)
	assert.False(t, ok)
}

func TestRegistry_ListSorted(t *testing.T) {
	r := NewRegistry()

	require.NoError(t, r.Register(Descriptor{Type: CapKanban, Backend: "kanboard"}))
	require.NoError(t, r.Register(Descriptor{Type: CapArchive, Backend: "archivebox"}))
	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep"}))

	list := r.List()
	require.Len(t, list, 3)
	assert.Equal(t, CapArchive, list[0].Type)
	assert.Equal(t, CapBookmark, list[1].Type)
	assert.Equal(t, CapKanban, list[2].Type)
}

func TestRegistry_ListEmpty(t *testing.T) {
	r := NewRegistry()

	list := r.List()
	assert.Empty(t, list)
}

func TestRegistry_RegisterWithOperations(t *testing.T) {
	r := NewRegistry()

	desc := Descriptor{
		Type:    CapBookmark,
		Backend: "karakeep",
		App:     "karakeep",
		Operations: []Operation{
			{
				Name:        "list",
				Description: "List all bookmarks",
				Input:       []ParamDef{{Name: "limit", Type: "int"}},
				Output:      []ParamDef{{Name: "items", Type: "array"}},
				Scopes:      []string{"read"},
			},
		},
	}
	require.NoError(t, r.Register(desc))

	got, ok := r.Get(CapBookmark)
	assert.True(t, ok)
	require.Len(t, got.Operations, 1)
	assert.Equal(t, "list", got.Operations[0].Name)
	assert.Equal(t, "read", got.Operations[0].Scopes[0])
}

func TestRegistry_RegisterMultipleCapabilities(t *testing.T) {
	r := NewRegistry()

	require.NoError(t, r.Register(Descriptor{Type: CapBookmark, Backend: "karakeep"}))
	require.NoError(t, r.Register(Descriptor{Type: CapArchive, Backend: "archivebox"}))
	require.NoError(t, r.Register(Descriptor{Type: CapReader, Backend: "miniflux"}))
	require.NoError(t, r.Register(Descriptor{Type: CapKanban, Backend: "kanboard"}))
	require.NoError(t, r.Register(Descriptor{Type: CapFinance, Backend: "fireflyiii"}))
	require.NoError(t, r.Register(Descriptor{Type: CapInfra, Backend: "beszel"}))
	require.NoError(t, r.Register(Descriptor{Type: CapShellHistory, Backend: "atuin"}))

	list := r.List()
	assert.Len(t, list, 7)
}

func TestDefaultRegistryIsNotNil(t *testing.T) {
	assert.NotNil(t, Default)
}

func TestDescriptorZeroValue(t *testing.T) {
	var d Descriptor
	assert.Empty(t, d.Type)
	assert.Empty(t, d.Backend)
	assert.Empty(t, d.App)
	assert.False(t, d.Healthy)
	assert.Nil(t, d.Instance)
	assert.Nil(t, d.Operations)
}
