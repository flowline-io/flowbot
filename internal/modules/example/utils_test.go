package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    int64
		b    int64
		want int64
	}{
		{name: "positive numbers", a: 1, b: 2, want: 3},
		{name: "zero plus zero", a: 0, b: 0, want: 0},
		{name: "positive plus negative", a: 1, b: -2, want: -1},
		{name: "negatives cancel out", a: -5, b: 5, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, add(tt.a, tt.b))
		})
	}
}
