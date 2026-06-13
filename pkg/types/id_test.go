package types

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeShortUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		uuid      string
		wantShort string
	}{
		{
			name:      "all zeros",
			uuid:      "00000000-0000-0000-0000-000000000000",
			wantShort: "2222222222222222222222",
		},
		{
			name:      "sample uuid",
			uuid:      "0026636a-e9b3-4a88-9c66-bf49d8cad81f",
			wantShort: "23XgxsETJcF7vB5N3FZRcB",
		},
		{
			name:      "another sample uuid",
			uuid:      "f9ee01c3-2015-4716-930e-4d5449810833",
			wantShort: "nUfojcH2M5j9j3Tk5A8mf7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, err := uuid.Parse(tt.uuid)
			require.NoError(t, err)
			assert.Equal(t, tt.wantShort, encodeShortUUID(u))
		})
	}
}

func TestId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "length is 22 chars"},
		{name: "ids are unique across calls"},
		{name: "ids use base57 alphabet only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id := Id()
			assert.Len(t, id, idEncLen)
			assert.NotContains(t, id, "cron:")
			assert.NotContains(t, id, "webhook:")

			for _, ch := range id {
				assert.True(t, strings.ContainsRune(idAlphabet, ch))
			}

			id2 := Id()
			assert.Len(t, id2, idEncLen)
			assert.NotEqual(t, id, id2)
		})
	}
}
