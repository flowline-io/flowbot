package utils

import (
	"testing"

	"github.com/bytedance/sonic"
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

func FuzzSameStringSlice(f *testing.F) {
	f.Add([]byte(`[]`), []byte(`[]`))
	f.Add([]byte(`["a"]`), []byte(`["a"]`))
	f.Add([]byte(`["a","b","c"]`), []byte(`["c","b","a"]`))
	f.Add([]byte(`["a"]`), []byte(`["b"]`))

	f.Fuzz(func(t *testing.T, xData, yData []byte) {
		var x, y []string
		if err := sonic.Unmarshal(xData, &x); err != nil {
			t.Skip()
		}
		if err := sonic.Unmarshal(yData, &y); err != nil {
			t.Skip()
		}

		if recovered := safeCall(func() { SameStringSlice(x, y) }); recovered != nil {
			t.Fatalf("SameStringSlice panicked: %v", recovered)
		}

		if !SameStringSlice(x, x) {
			t.Errorf("not reflexive: x=%v", x)
		}

		if SameStringSlice(x, y) != SameStringSlice(y, x) {
			t.Errorf("not symmetric: x=%v, y=%v", x, y)
		}
	})
}

func safeCall(fn func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	fn()
	return
}
