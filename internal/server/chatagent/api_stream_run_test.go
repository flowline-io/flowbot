package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type streamCapture struct {
	events []StreamEvent
}

func (c *streamCapture) WriteEvent(event StreamEvent) bool {
	c.events = append(c.events, event)
	return event.Type == EventTypeDone ||
		event.Type == EventTypeError ||
		event.Type == EventTypeCanceled
}

func TestStreamAPIRun_CompletesWithFakeModel(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "stream reply")

	ctx := context.Background()
	sessionID := "sess-stream-run"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user-1", State: int(schema.ChatSessionActive),
	}))

	captured := &streamCapture{}
	StreamAPIRun(ctx, NewService(), sessionID, "hello stream", captured)
	WaitForSessionTitleGenerationForTest()

	require.NotEmpty(t, captured.events)
	last := captured.events[len(captured.events)-1]
	assert.Equal(t, EventTypeDone, last.Type)
}

func TestStreamAPIRun_EmptyMessageReturnsError(t *testing.T) {
	LockAppConfigForTest(t)
	t.Cleanup(DisableSessionTitleLLMForTest())
	setupEphemeralRunTestDB(t)
	setupEphemeralRunFakeModel(t, "unused")

	ctx := context.Background()
	sessionID := "sess-stream-empty"
	require.NoError(t, store.Database.CreateChatSession(ctx, &gen.ChatSession{
		Flag: sessionID, UID: "user-1", State: int(schema.ChatSessionActive),
	}))

	captured := &streamCapture{}
	StreamAPIRun(ctx, NewService(), sessionID, "   ", captured)

	require.NotEmpty(t, captured.events)
	assert.Equal(t, EventTypeError, captured.events[len(captured.events)-1].Type)
}
