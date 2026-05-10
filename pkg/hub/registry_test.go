package hub

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestNewRegistry(t *testing.T) {
	t.Run("creates non-nil registry with empty list", func(t *testing.T) {
		r := NewRegistry()
		require.NotNil(t, r)
		assert.Empty(t, r.List())
	})
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name        string
		descriptors []Descriptor
		wantErr     bool
		errContains string
		errIs       error
		check       func(*testing.T, *Registry)
	}{
		{
			name: "registers descriptor successfully",
			descriptors: []Descriptor{
				{
					Type:        CapBookmark,
					Backend:     "karakeep",
					App:         "karakeep",
					Description: "Bookmark service",
					Healthy:     true,
				},
			},
			check: func(t *testing.T, r *Registry) {
				got, ok := r.Get(CapBookmark)
				assert.True(t, ok)
				assert.Equal(t, CapBookmark, got.Type)
				assert.Equal(t, "karakeep", got.Backend)
				assert.Equal(t, "karakeep", got.App)
				assert.True(t, got.Healthy)
			},
		},
		{
			name: "empty capability type",
			descriptors: []Descriptor{
				{Backend: "test", App: "test"},
			},
			wantErr:     true,
			errContains: "capability type is required",
			errIs:       types.ErrInvalidArgument,
		},
		{
			name: "overwrites existing capability",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep", App: "karakeep"},
				{Type: CapBookmark, Backend: "linkwarden", App: "linkwarden"},
			},
			check: func(t *testing.T, r *Registry) {
				got, ok := r.Get(CapBookmark)
				assert.True(t, ok)
				assert.Equal(t, "linkwarden", got.Backend)
			},
		},
		{
			name: "with operations",
			descriptors: []Descriptor{
				{
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
				},
			},
			check: func(t *testing.T, r *Registry) {
				got, ok := r.Get(CapBookmark)
				assert.True(t, ok)
				require.Len(t, got.Operations, 1)
				assert.Equal(t, "list", got.Operations[0].Name)
				assert.Equal(t, "read", got.Operations[0].Scopes[0])
			},
		},
		{
			name: "multiple capabilities",
			descriptors: []Descriptor{
				{Type: CapBookmark, Backend: "karakeep"},
				{Type: CapArchive, Backend: "archivebox"},
				{Type: CapReader, Backend: "miniflux"},
				{Type: CapKanban, Backend: "kanboard"},
				{Type: CapFinance, Backend: "fireflyiii"},
				{Type: CapInfra, Backend: "beszel"},
				{Type: CapShellHistory, Backend: "atuin"},
			},
			check: func(t *testing.T, r *Registry) {
				list := r.List()
				assert.Len(t, list, 7)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			var lastErr error
			for i, d := range tt.descriptors {
				lastErr = r.Register(d)
				if tt.wantErr && i == 0 {
					break
				}
				require.NoError(t, lastErr)
			}
			if tt.wantErr {
				require.Error(t, lastErr)
				if tt.errContains != "" {
					assert.Contains(t, lastErr.Error(), tt.errContains)
				}
				if tt.errIs != nil {
					assert.True(t, errors.Is(lastErr, tt.errIs))
				}
			}
			if tt.check != nil {
				tt.check(t, r)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	tests := []struct {
		name    string
		capType CapabilityType
		wantOk  bool
	}{
		{
			name:    "missing capability",
			capType: CapArchive,
			wantOk:  false,
		},
		{
			name:    "empty registry returns false",
			capType: CapReader,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			_, ok := r.Get(tt.capType)
			assert.False(t, ok)
		})
	}
}

func TestRegistry_List(t *testing.T) {
	tests := []struct {
		name        string
		descriptors []Descriptor
		check       func(*testing.T, []Descriptor)
	}{
		{
			name: "sorted by capability type",
			descriptors: []Descriptor{
				{Type: CapKanban, Backend: "kanboard"},
				{Type: CapArchive, Backend: "archivebox"},
				{Type: CapBookmark, Backend: "karakeep"},
			},
			check: func(t *testing.T, list []Descriptor) {
				require.Len(t, list, 3)
				assert.Equal(t, CapArchive, list[0].Type)
				assert.Equal(t, CapBookmark, list[1].Type)
				assert.Equal(t, CapKanban, list[2].Type)
			},
		},
		{
			name:        "empty registry",
			descriptors: nil,
			check: func(t *testing.T, list []Descriptor) {
				assert.Empty(t, list)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			for _, d := range tt.descriptors {
				require.NoError(t, r.Register(d))
			}
			list := r.List()
			tt.check(t, list)
		})
	}
}

func TestDefaultRegistryIsNotNil(t *testing.T) {
	t.Run("default registry is not nil", func(t *testing.T) {
		assert.NotNil(t, Default)
	})
}

func TestDescriptorZeroValue(t *testing.T) {
	t.Run("descriptor fields are zero values", func(t *testing.T) {
		var d Descriptor
		assert.Empty(t, d.Type)
		assert.Empty(t, d.Backend)
		assert.Empty(t, d.App)
		assert.False(t, d.Healthy)
		assert.Nil(t, d.Instance)
		assert.Nil(t, d.Operations)
	})
}
