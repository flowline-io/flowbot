package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item present",
			slice: []string{"/livez", "/readyz", "/metrics"},
			item:  "/metrics",
			want:  true,
		},
		{
			name:  "item absent",
			slice: []string{"/livez", "/readyz"},
			item:  "/healthz",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "/",
			want:  false,
		},
		{
			name:  "nil slice",
			slice: nil,
			item:  "/",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Contains(tt.slice, tt.item))
		})
	}
}

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
		{
			name: "nil slices",
			x:    nil,
			y:    nil,
			want: true,
		},
		{
			name: "duplicate mismatch",
			x:    []string{"a", "a"},
			y:    []string{"a", "b"},
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
