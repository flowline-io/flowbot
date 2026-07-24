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
				writeStreamEventsUntilRunDone(captured, pub, runDone, nil)
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

func TestWriteStreamEventsUntilRunDone_DetachesOnTerminalWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		eventType string
	}{
		{name: "detaches on done", eventType: EventTypeDone},
		{name: "detaches on error", eventType: EventTypeError},
		{name: "detaches on canceled", eventType: EventTypeCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := NewChannelPublisher(8)
			runDone := make(chan error, 1)
			detached := make(chan struct{}, 1)
			require.NoError(t, pub.Publish(StreamEvent{Type: tt.eventType, Message: "x"}))
			go func() {
				time.Sleep(20 * time.Millisecond)
				pub.Close()
				runDone <- nil
			}()

			writeStreamEventsUntilRunDone(&streamCapture{}, pub, runDone, func() {
				select {
				case detached <- struct{}{}:
				default:
				}
			})

			select {
			case <-detached:
			default:
				t.Fatal("onDetach was not called after terminal write")
			}
		})
	}
}

type failAfterWriter struct {
	mu     sync.Mutex
	after  int
	writes int
	events []StreamEvent
}

func (f *failAfterWriter) WriteEvent(event StreamEvent) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	// Always accept terminal events so we can assert Done survives mid-stream I/O failure.
	if isTerminalStreamEvent(event) {
		f.events = append(f.events, event)
		return true
	}
	if f.after >= 0 && f.writes > f.after {
		return true
	}
	f.events = append(f.events, event)
	return false
}

func (f *failAfterWriter) gotTypes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.events))
	for i, ev := range f.events {
		out[i] = ev.Type
	}
	return out
}

func TestWriteStreamEventsUntilRunDone_IOFailureStillDeliversDone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		failAfter int
		terminal  StreamEvent
		wantType  string
	}{
		{
			name:      "done after mid-stream write failure",
			failAfter: 1,
			terminal:  StreamEvent{Type: EventTypeDone, Text: "final reply"},
			wantType:  EventTypeDone,
		},
		{
			name:      "error after mid-stream write failure",
			failAfter: 1,
			terminal:  StreamEvent{Type: EventTypeError, Message: "boom"},
			wantType:  EventTypeError,
		},
		{
			name:      "canceled after first-write failure",
			failAfter: 0,
			terminal:  StreamEvent{Type: EventTypeCanceled, Message: "stopped"},
			wantType:  EventTypeCanceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hub := &SessionEventHub{subs: make(map[string]*ChannelPublisher)}
			pub := hub.Subscribe("run", 8)
			runDone := make(chan error, 1)
			writer := &failAfterWriter{after: tt.failAfter}

			done := make(chan struct{})
			go func() {
				writeStreamEventsUntilRunDone(writer, pub, runDone, func() {
					hub.Detach("run")
				})
				close(done)
			}()

			require.NoError(t, pub.Publish(StreamEvent{Type: EventTypeDelta, Text: "a"}))
			if tt.failAfter >= 1 {
				require.NoError(t, pub.Publish(StreamEvent{Type: EventTypeDelta, Text: "b"}))
			}
			observer := hub.Subscribe("events", 4)
			hub.publish(StreamEvent{Type: EventTypeConfirmResolved, ID: "c1", Approved: true})
			select {
			case ev := <-observer.Events():
				assert.Equal(t, EventTypeConfirmResolved, ev.Type)
			case <-time.After(time.Second):
				t.Fatal("observer did not receive confirm_resolved after detach")
			}

			require.NoError(t, pub.Publish(tt.terminal))
			pub.Close()
			runDone <- nil

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("writeStreamEventsUntilRunDone did not return")
			}

			types := writer.gotTypes()
			require.Contains(t, types, tt.wantType, "terminal event must survive mid-stream I/O failure")
		})
	}
}

func TestHubDetachDoesNotClosePublisher(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, hub *SessionEventHub)
	}{
		{
			name: "publish works after detach",
			run: func(t *testing.T, hub *SessionEventHub) {
				pub := hub.Subscribe("run", 4)
				hub.Detach("run")
				require.NoError(t, pub.Publish(StreamEvent{Type: EventTypeDone, Text: "ok"}))
				select {
				case ev := <-pub.Events():
					assert.Equal(t, EventTypeDone, ev.Type)
				case <-time.After(time.Second):
					t.Fatal("publisher closed by Detach")
				}
			},
		},
		{
			name: "unsubscribe still closes",
			run: func(t *testing.T, hub *SessionEventHub) {
				pub := hub.Subscribe("events", 4)
				hub.Unsubscribe("events")
				require.NoError(t, pub.Publish(StreamEvent{Type: EventTypeDelta, Text: "x"}))
				select {
				case _, ok := <-pub.Events():
					assert.False(t, ok, "Unsubscribe should close publisher")
				case <-time.After(50 * time.Millisecond):
					t.Fatal("expected closed channel after Unsubscribe")
				}
			},
		},
		{
			name: "detach is idempotent",
			run: func(t *testing.T, hub *SessionEventHub) {
				pub := hub.Subscribe("run", 4)
				hub.Detach("run")
				hub.Detach("run")
				require.NoError(t, pub.Publish(StreamEvent{Type: EventTypeDelta, Text: "still-open"}))
				select {
				case ev := <-pub.Events():
					assert.Equal(t, EventTypeDelta, ev.Type)
				case <-time.After(time.Second):
					t.Fatal("publisher closed by repeated Detach")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hub := &SessionEventHub{subs: make(map[string]*ChannelPublisher)}
			tt.run(t, hub)
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
	NewService().StreamAPIRun(ctx, sessionID, "hello stream", nil, "", captured)
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
	NewService().StreamAPIRun(ctx, sessionID, "   ", nil, "", captured)

	events := captured.snapshot()
	require.NotEmpty(t, events)
	assert.Equal(t, EventTypeError, events[len(events)-1].Type)
	assert.Contains(t, events[len(events)-1].Message, "empty message")
}
