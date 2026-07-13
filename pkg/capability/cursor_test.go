package capability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		capability string
		backend    string
		strategy   string
		page       int
		limit      int
		offset     int
		sortBy     string
		sortOrder  string
	}{
		{"encode and decode cursor returns original payload", "karakeep", "karakeep", "page", 2, 20, 0, "", ""},
		{"round trip with offset-based strategy preserves offset", "miniflux", "miniflux", "offset", 0, 30, 50, "id", "asc"},
		{"round trip with sort fields preserves sort info", "kanboard", "kanboard", "page", 1, 10, 0, "created_at", "desc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			now := time.Unix(1700000000, 0)
			cursor, err := EncodeCursor([]byte("secret"), CursorPayload{
				Capability: tt.capability,
				Backend:    tt.backend,
				Strategy:   tt.strategy,
				Page:       tt.page,
				Limit:      tt.limit,
				Offset:     tt.offset,
				SortBy:     tt.sortBy,
				SortOrder:  tt.sortOrder,
				ExpiresAt:  now.Add(time.Hour),
			})
			require.NoError(t, err)

			payload, err := DecodeCursor([]byte("secret"), cursor, now)
			require.NoError(t, err)
			require.Equal(t, tt.capability, payload.Capability)
			require.Equal(t, tt.backend, payload.Backend)
			require.Equal(t, tt.strategy, payload.Strategy)
			require.Equal(t, tt.page, payload.Page)
			require.Equal(t, tt.limit, payload.Limit)
			require.Equal(t, tt.offset, payload.Offset)
			require.Equal(t, tt.sortBy, payload.SortBy)
			require.Equal(t, tt.sortOrder, payload.SortOrder)
		})
	}
}

func TestCursorRejectsTampering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		corruptFunc func(string) string
	}{
		{"tampered cursor returns invalid argument error", func(c string) string { return c + "x" }},
		{"cursor with truncated signature returns invalid argument error", func(c string) string { return c[:len(c)-5] }},
		{"cursor with wrong secret returns invalid argument error", func(c string) string { return c }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			now := time.Unix(1700000000, 0)
			secret := []byte("secret")
			if tt.name == "cursor with wrong secret returns invalid argument error" {
				secret = []byte("different-secret")
			}
			cursor, err := EncodeCursor([]byte("secret"), CursorPayload{Capability: "karakeep"})
			require.NoError(t, err)

			_, err = DecodeCursor(secret, tt.corruptFunc(cursor), now)
			require.Error(t, err)
			require.ErrorIs(t, err, types.ErrInvalidArgument)
		})
	}
}

func TestCursorRejectsExpired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		expires time.Time
	}{
		{"expired cursor returns error containing expired", time.Unix(1700000000, 0).Add(-time.Second)},
		{"barely expired by one second returns error containing expired", time.Unix(1700000000, 0).Add(-time.Second)},
		{"deeply expired cursor returns error containing expired", time.Unix(1700000000, 0).Add(-365 * 24 * time.Hour)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			now := time.Unix(1700000000, 0)
			cursor, err := EncodeCursor([]byte("secret"), CursorPayload{ExpiresAt: tt.expires})
			require.NoError(t, err)

			_, err = DecodeCursor([]byte("secret"), cursor, now)
			require.Error(t, err)
			require.Contains(t, err.Error(), "expired")
		})
	}
}
