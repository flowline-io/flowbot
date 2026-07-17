package chatagent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type streamCapture struct {
	mu     sync.Mutex
	events []StreamEvent
}

func (c *streamCapture) WriteEvent(event StreamEvent) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
	return event.Type == EventTypeDone ||
		event.Type == EventTypeError ||
		event.Type == EventTypeCanceled
}

func (c *streamCapture) snapshot() []StreamEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]StreamEvent, len(c.events))
	copy(out, c.events)
	return out
}

func TestWriteStreamEventsUntilRunDone(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, pub *ChannelPublisher, runDone chan error)
		wantType string
		wantMsg  string
	}{
		{
			name: "publisher close without events still writes run error",
			setup: func(t *testing.T, pub *ChannelPublisher, runDone chan error) {
				t.Helper()
				pub.Close()
				runDone <- errors.New("empty message")
			},
			wantType: EventTypeError,
			wantMsg:  "empty message",
		},
		{
			name: "done event waits for runDone before returning",
			setup: func(t *testing.T, pub *ChannelPublisher, runDone chan error) {
				t.Helper()
				require.NoError(t, pub.Publish(StreamEvent{Type: EventTypeDone, Text: "ok"}))
				go func() {
					time.Sleep(20 * time.Millisecond)
					pub.Close()
					runDone <- nil
				}()
			},
			wantType: EventTypeDone,
		},
		{
			name: "canceled run writes canceled event",
			setup: func(t *testing.T, pub *ChannelPublisher, runDone chan error) {
				t.Helper()
				pub.Close()
				runDone <- context.Canceled
			},
			wantType: EventTypeCanceled,
			wantMsg:  "run canceled by user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := NewChannelPublisher(8)
			runDone := make(chan error, 1)
			tt.setup(t, pub, runDone)

			captured := &streamCapture{}
			done := make(chan struct{})
			go func() {
				writeStreamEventsUntilRunDone(captured, pub, runDone)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("writeStreamEventsUntilRunDone did not return")
			}

			events := captured.snapshot()
			require.NotEmpty(t, events)
			last := events[len(events)-1]
			assert.Equal(t, tt.wantType, last.Type)
			if tt.wantMsg != "" {
				assert.Equal(t, tt.wantMsg, last.Message)
			}
		})
	}
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

	events := captured.snapshot()
	require.NotEmpty(t, events)
	last := events[len(events)-1]
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

	events := captured.snapshot()
	require.NotEmpty(t, events)
	assert.Equal(t, EventTypeError, events[len(events)-1].Type)
	assert.Contains(t, events[len(events)-1].Message, "empty message")
}
