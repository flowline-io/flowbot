package chatagent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestResolveDeliveryContextIncludesThreadID(t *testing.T) {
	tests := []struct {
		name       string
		content    map[string]any
		wantThread string
	}{
		{
			name:       "reads thread_id from message content",
			content:    map[string]any{"text": "hello", "thread_id": "1700000000.000100"},
			wantThread: "1700000000.000100",
		},
		{
			name:       "missing thread_id stays empty",
			content:    map[string]any{"text": "hello"},
			wantThread: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEphemeralRunTestDB(t)
			ctx := context.Background()
			sessionID := types.Id()
			require.NoError(t, store.ChatStoreFromDB().CreateChatSession(ctx, &gen.ChatSession{
				Flag:  sessionID,
				UID:   "user-1",
				State: int(schema.ChatSessionActive),
			}))
			platformID, err := store.PlatformStoreFromDB().CreatePlatform(ctx, &gen.Platform{Name: "slack"})
			require.NoError(t, err)
			require.NoError(t, store.MessageStoreFromDB().CreateMessage(ctx, gen.Message{
				Flag:       types.Id(),
				PlatformID: platformID,
				Topic:      "D123",
				Role:       types.User,
				Session:    sessionID,
				Content:    tt.content,
				State:      int(schema.MessageCreated),
			}))

			got := ResolveDeliveryContext(ctx, sessionID)
			assert.Equal(t, "D123", got.Topic)
			assert.Equal(t, "slack", got.Platform)
			assert.Equal(t, tt.wantThread, got.ThreadID)
		})
	}
}

func TestDeliveryToMapRoundTripThreadID(t *testing.T) {
	in := ScheduledDelivery{Platform: "slack", Topic: "D1", PlatformID: 9, ThreadID: "1700000000.000200"}
	m := deliveryToMap(in)
	require.NotNil(t, m)
	assert.Equal(t, "1700000000.000200", m["thread_id"])

	out := deliveryFromTask(&gen.ChatScheduledTask{Delivery: m})
	assert.Equal(t, in, out)
}
