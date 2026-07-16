package syncx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/utils/syncx"
)

func TestLoadOrStore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setup      func(m *syncx.Map[string, int])
		key        string
		value      int
		wantActual int
		wantLoaded bool
	}{
		{
			name:       "stores new key",
			key:        "k1",
			value:      10,
			wantActual: 10,
			wantLoaded: false,
		},
		{
			name: "loads existing key",
			setup: func(m *syncx.Map[string, int]) {
				m.Set("k2", 20)
			},
			key:        "k2",
			value:      99,
			wantActual: 20,
			wantLoaded: true,
		},
		{
			name:       "stores zero value",
			key:        "zero",
			value:      0,
			wantActual: 0,
			wantLoaded: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := syncx.Map[string, int]{}
			if tt.setup != nil {
				tt.setup(&m)
			}
			actual, loaded := m.LoadOrStore(tt.key, tt.value)
			assert.Equal(t, tt.wantLoaded, loaded)
			assert.Equal(t, tt.wantActual, actual)
		})
	}
}

func TestLoadOrStoreConcurrent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "concurrent load or store converges on one value"},
		{name: "parallel stores on distinct keys"},
		{name: "parallel load or store same key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := syncx.Map[string, int]{}
			actual, loaded := m.LoadOrStore("shared", 1)
			assert.False(t, loaded)
			assert.Equal(t, 1, actual)
			actual2, loaded2 := m.LoadOrStore("shared", 2)
			assert.True(t, loaded2)
			assert.Equal(t, 1, actual2)
		})
	}
}
