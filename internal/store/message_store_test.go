package store

import (
	"context"
	"testing"
	"time"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		run  func(*testing.T, *gen.Client)
	}{
		{
			name: "create get by flag platform and session",
			run: func(t *testing.T, client *gen.Client) {
				require.NoError(t, NewMessageStore(client).CreateMessage(ctx, gen.Message{
					Flag: "msg-1", PlatformID: 1, PlatformMsgID: "pmsg-1",
					Topic: "general", Role: "user", Session: "sess-1",
					State: int(schema.MessageCreated), Content: map[string]any{"text": "hi"},
					CreatedAt: now, UpdatedAt: now,
				}))
				got, err := NewMessageStore(client).GetMessage(ctx, "msg-1")
				require.NoError(t, err)
				assert.Equal(t, "hi", got.Content["text"])

				byPlatform, err := NewMessageStore(client).GetMessageByPlatform(ctx, 1, "pmsg-1")
				require.NoError(t, err)
				assert.Equal(t, "msg-1", byPlatform.Flag)

				rows, err := NewMessageStore(client).GetMessagesBySession(ctx, "sess-1")
				require.NoError(t, err)
				require.Len(t, rows, 1)
			},
		},
		{
			name: "get missing message returns not found",
			run: func(t *testing.T, client *gen.Client) {
				_, err := NewMessageStore(client).GetMessage(ctx, "missing")
				require.ErrorIs(t, err, types.ErrNotFound)
			},
		},
		{
			name: "empty session returns empty slice",
			run: func(t *testing.T, client *gen.Client) {
				rows, err := NewMessageStore(client).GetMessagesBySession(ctx, "empty")
				require.NoError(t, err)
				assert.Empty(t, rows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, sqlitetest.OpenClient(t, t.Name()))
		})
	}
}
