package server

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneProtectsSessionIDFromBufferReuse(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		overwrite string
		wantCorrupt string
	}{
		{
			name:        "render-markdown overwrites prefix leaving session suffix",
			sessionID:   "CevXBUi8KZLY4oW7Hi4BPL",
			overwrite:   "render-markdown",
			wantCorrupt: "render-markdown7Hi4BPL",
		},
		{
			name:        "shorter path segment leaves trailing bytes",
			sessionID:   "Wfm2Fx4vSzBz9z3Cbt2eeZ",
			overwrite:   "render-markdown",
			wantCorrupt: "render-markdownCbt2eeZ",
		},
		{
			name:        "exact length overwrite replaces fully",
			sessionID:   "abcdefghijklmno",
			overwrite:   "render-markdown",
			wantCorrupt: "render-markdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := []byte(tt.sessionID)
			shared := unsafe.String(unsafe.SliceData(buf), len(buf))
			cloned := strings.Clone(shared)

			require.Equal(t, tt.sessionID, shared)
			require.Equal(t, tt.sessionID, cloned)

			copy(buf, tt.overwrite)
			assert.Equal(t, tt.wantCorrupt, shared)
			assert.Equal(t, tt.sessionID, cloned)
		})
	}
}
