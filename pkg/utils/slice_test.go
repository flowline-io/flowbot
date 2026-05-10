package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSameStringSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		x    []string
		y    []string
		want bool
	}{
		{
			name: "equal sets",
			x:    []string{"a", "b", "c", "d", "e"},
			y:    []string{"d", "a", "e", "b", "c"},
			want: true,
		},
		{
			name: "different sets",
			x:    []string{"a", "b", "c", "d", "e"},
			y:    []string{"d", "a", "f", "b", "c"},
			want: false,
		},
		{
			name: "empty slices",
			x:    []string{},
			y:    []string{},
			want: true,
		},
		{
			name: "one empty",
			x:    []string{"a"},
			y:    []string{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SameStringSlice(tt.x, tt.y)
			assert.Equal(t, tt.want, got)
		})
	}
}
