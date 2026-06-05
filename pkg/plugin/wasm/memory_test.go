package wasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   uint64
		wantPtr  uint32
		wantSize uint32
	}{
		{
			name:     "zero result",
			result:   0,
			wantPtr:  0,
			wantSize: 0,
		},
		{
			name:     "normal result",
			result:   (1024 << 32) | 256,
			wantPtr:  1024,
			wantSize: 256,
		},
		{
			name:     "max uint32 values",
			result:   (4294967295 << 32) | 4294967295,
			wantPtr:  4294967295,
			wantSize: 4294967295,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr, size := decodeResult(tt.result)
			assert.Equal(t, tt.wantPtr, ptr)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}
