package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	t.Run("creates non-nil registry with empty list", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NotNil(t, r)
		assert.Empty(t, r.List())
	})
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

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
					Type:        CapKarakeep,
					App:         "karakeep",
					Description: "Bookmark service",
					Healthy:     true,
				},
			},
			check: func(t *testing.T, r *Registry) {
				got, ok := r.Get(CapKarakeep)
				assert.True(t, ok)
				assert.Equal(t, CapKarakeep, got.Type)
				assert.Equal(t, "karakeep", got.App)
				assert.True(t, got.Healthy)
			},
		},
		{
			name: "empty capability type",
			descriptors: []Descriptor{
				{App: "test"},
			},
			wantErr:     true,
			errContains: "capability type is required",
			errIs:       types.ErrInvalidArgument,
		},
		{
			name: "overwrites existing capability",
			descriptors: []Descriptor{
				{Type: CapKarakeep, App: "karakeep"},
				{Type: CapKarakeep, App: "linkwarden"},
			},
			check: func(t *testing.T, r *Registry) {
				got, ok := r.Get(CapKarakeep)
				assert.True(t, ok)
				assert.Equal(t, "linkwarden", got.App)
			},
		},
		{
			name: "with operations",
			descriptors: []Descriptor{
				{
					Type: CapKarakeep,
					App:  "karakeep",
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
				got, ok := r.Get(CapKarakeep)
				assert.True(t, ok)
				require.Len(t, got.Operations, 1)
				assert.Equal(t, "list", got.Operations[0].Name)
				assert.Equal(t, "read", got.Operations[0].Scopes[0])
			},
		},
		{
			name: "multiple capabilities",
			descriptors: []Descriptor{
				{Type: CapKarakeep},
				{Type: CapExample},
				{Type: CapMiniflux},
				{Type: CapKanboard},
				{Type: CapTrilium},
				{Type: CapMemos},
				{Type: CapGitea},
			},
			check: func(t *testing.T, r *Registry) {
				list := r.List()
				assert.Len(t, list, 7)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
					require.ErrorIs(t, lastErr, tt.errIs)
				}
			}
			if tt.check != nil {
				tt.check(t, r)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		capType CapabilityType
		wantOk  bool
	}{
		{
			name:    "missing capability",
			capType: CapExample,
			wantOk:  false,
		},
		{
			name:    "empty registry returns false",
			capType: CapMiniflux,
			wantOk:  false,
		},
		{
			name:    "retrieves registered capability",
			capType: CapKarakeep,
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			if tt.wantOk {
				require.NoError(t, r.Register(Descriptor{Type: tt.capType}))
			}
			_, ok := r.Get(tt.capType)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		descriptors []Descriptor
		check       func(*testing.T, []Descriptor)
	}{
		{
			name: "sorted by capability type",
			descriptors: []Descriptor{
				{Type: CapKanboard},
				{Type: CapExample},
				{Type: CapKarakeep},
			},
			check: func(t *testing.T, list []Descriptor) {
				require.Len(t, list, 3)
				assert.Equal(t, CapExample, list[0].Type)
				assert.Equal(t, CapKanboard, list[1].Type)
				assert.Equal(t, CapKarakeep, list[2].Type)
			},
		},
		{
			name:        "empty registry",
			descriptors: nil,
			check: func(t *testing.T, list []Descriptor) {
				assert.Empty(t, list)
			},
		},
		{
			name: "single capability",
			descriptors: []Descriptor{
				{Type: CapKarakeep, App: "karakeep"},
			},
			check: func(t *testing.T, list []Descriptor) {
				require.Len(t, list, 1)
				assert.Equal(t, CapKarakeep, list[0].Type)
				assert.Equal(t, "karakeep", list[0].App)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
		assert.Empty(t, d.App)
		assert.False(t, d.Healthy)
		assert.Nil(t, d.Instance)
		assert.Nil(t, d.Operations)
	})
}
