package chatagent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelPublisherPublish(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
	}{
		{name: "drops delta when full", eventType: EventTypeDelta},
		{name: "drops confirm when full", eventType: EventTypeConfirm},
		{name: "drops done when full", eventType: EventTypeDone},
		{name: "drops mode_change when full", eventType: EventTypeModeChange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := NewChannelPublisher(1)
			require.NoError(t, pub.Publish(StreamEvent{Type: EventTypeDelta, Text: "first"}))

			done := make(chan struct{})
			go func() {
				_ = pub.Publish(StreamEvent{Type: tt.eventType, Text: "second"})
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(50 * time.Millisecond):
				t.Fatal("publish should return immediately when full")
			}
			ev := <-pub.Events()
			assert.Equal(t, "first", ev.Text)
			select {
			case extra := <-pub.Events():
				t.Fatalf("unexpected extra event: %+v", extra)
			default:
			}
		})
	}
}

func TestChannelPublisherTimingEventsNonBlocking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event StreamEvent
	}{
		{
			name:  "turn drops when full",
			event: StreamEvent{Type: EventTypeTurn, DurationMs: 900, Step: 1},
		},
		{
			name:  "tool completed drops when full",
			event: StreamEvent{Type: EventTypeTool, Name: "echo", Status: "completed", DurationMs: 120},
		},
		{
			name:  "thinking completed drops when full",
			event: StreamEvent{Type: EventTypeThinking, Status: "completed", DurationMs: 300},
		},
		{
			name:  "tool running drops when full",
			event: StreamEvent{Type: EventTypeTool, Name: "echo", Status: "running"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := NewChannelPublisher(1)
			require.NoError(t, pub.Publish(StreamEvent{Type: EventTypeDelta, Text: "first"}))

			done := make(chan struct{})
			go func() {
				_ = pub.Publish(tt.event)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(50 * time.Millisecond):
				t.Fatal("publish should return immediately when full")
			}
		})
	}
}

func TestSessionEventHubPublishDoesNotBlockOnFullSubscriber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		eventType string
	}{
		{name: "confirm reaches live observer while abandoned run buffer is full", eventType: EventTypeConfirm},
		{name: "run_complete reaches live observer while abandoned run buffer is full", eventType: EventTypeRunComplete},
		{name: "confirm_resolved reaches live observer while abandoned run buffer is full", eventType: EventTypeConfirmResolved},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hub := &SessionEventHub{subs: make(map[string]*ChannelPublisher)}
			abandoned := hub.Subscribe("run", 1)
			require.NoError(t, abandoned.Publish(StreamEvent{Type: EventTypeDelta, Text: "fill"}))

			observer := hub.Subscribe("events", 8)
			done := make(chan struct{})
			go func() {
				hub.publish(StreamEvent{Type: tt.eventType, ID: "c1", Approved: true})
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("hub publish blocked on abandoned subscriber")
			}

			select {
			case ev := <-observer.Events():
				assert.Equal(t, tt.eventType, ev.Type)
			case <-time.After(200 * time.Millisecond):
				t.Fatal("observer did not receive event")
			}
		})
	}
}
