package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestEventStore_DataEventExists(t *testing.T) {
	t.Parallel()
	st := NewEventStore(getTestClient(t))
	ctx := t.Context()

	require.NoError(t, st.AppendDataEvent(ctx, types.DataEvent{
		EventID:        "evt-image-1",
		EventType:      types.EventHomelabImageUpdateAvailable,
		IdempotencyKey: "karakeep/web/sha256:abc",
	}))

	tests := []struct {
		name string
		typ  string
		key  string
		want bool
	}{
		{name: "matching type and key", typ: types.EventHomelabImageUpdateAvailable, key: "karakeep/web/sha256:abc", want: true},
		{name: "same type different key", typ: types.EventHomelabImageUpdateAvailable, key: "karakeep/web/sha256:other", want: false},
		{name: "different type same key", typ: types.EventBookmarkCreated, key: "karakeep/web/sha256:abc", want: false},
		{name: "empty key is not found", typ: types.EventHomelabImageUpdateAvailable, key: "", want: false},
		{name: "empty type is not found", typ: "", key: "karakeep/web/sha256:abc", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := st.DataEventExists(ctx, tt.typ, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
