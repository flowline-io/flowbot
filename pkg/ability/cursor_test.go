package ability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestCursorRoundTrip(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"encode and decode cursor returns original payload"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Unix(1700000000, 0)
			cursor, err := EncodeCursor([]byte("secret"), CursorPayload{
				Capability: "bookmark",
				Backend:    "karakeep",
				Strategy:   "page",
				Page:       2,
				Limit:      20,
				ExpiresAt:  now.Add(time.Hour),
			})
			require.NoError(t, err)

			payload, err := DecodeCursor([]byte("secret"), cursor, now)
			require.NoError(t, err)
			require.Equal(t, "bookmark", payload.Capability)
			require.Equal(t, 2, payload.Page)
			require.Equal(t, 20, payload.Limit)
		})
	}
}

func TestCursorRejectsTampering(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"tampered cursor returns invalid argument error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Unix(1700000000, 0)
			cursor, err := EncodeCursor([]byte("secret"), CursorPayload{Capability: "bookmark"})
			require.NoError(t, err)

			_, err = DecodeCursor([]byte("secret"), cursor+"x", now)
			require.Error(t, err)
			require.ErrorIs(t, err, types.ErrInvalidArgument)
		})
	}
}

func TestCursorRejectsExpired(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"expired cursor returns error containing expired"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Unix(1700000000, 0)
			cursor, err := EncodeCursor([]byte("secret"), CursorPayload{ExpiresAt: now.Add(-time.Second)})
			require.NoError(t, err)

			_, err = DecodeCursor([]byte("secret"), cursor, now)
			require.Error(t, err)
			require.Contains(t, err.Error(), "expired")
		})
	}
}
