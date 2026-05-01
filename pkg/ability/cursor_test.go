package ability

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestCursorRoundTrip(t *testing.T) {
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
}

func TestCursorRejectsTampering(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cursor, err := EncodeCursor([]byte("secret"), CursorPayload{Capability: "bookmark"})
	require.NoError(t, err)

	_, err = DecodeCursor([]byte("secret"), cursor+"x", now)
	require.Error(t, err)
	require.True(t, errors.Is(err, types.ErrInvalidArgument))
}

func TestCursorRejectsExpired(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cursor, err := EncodeCursor([]byte("secret"), CursorPayload{ExpiresAt: now.Add(-time.Second)})
	require.NoError(t, err)

	_, err = DecodeCursor([]byte("secret"), cursor, now)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "expired"))
}
