package rdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKvHash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		item    any
		wantErr bool
		wantLen int
	}{
		{name: "hashes string map", item: map[string]string{"k": "v"}, wantLen: 64},
		{name: "hashes struct-like map", item: map[string]any{"id": 1, "name": "test"}, wantLen: 64},
		{name: "hashes slice", item: []int{1, 2, 3}, wantLen: 64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := kvHash(tt.item)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
			got2, err := kvHash(tt.item)
			require.NoError(t, err)
			assert.Equal(t, got, got2)
		})
	}
}

func TestKvHashDifferentInputs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    any
		b    any
	}{
		{name: "different strings produce different hashes", a: "alpha", b: "beta"},
		{name: "different maps produce different hashes", a: map[string]int{"a": 1}, b: map[string]int{"a": 2}},
		{name: "different types produce different hashes", a: "1", b: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ha, err := kvHash(tt.a)
			require.NoError(t, err)
			hb, err := kvHash(tt.b)
			require.NoError(t, err)
			assert.NotEqual(t, ha, hb)
		})
	}
}
