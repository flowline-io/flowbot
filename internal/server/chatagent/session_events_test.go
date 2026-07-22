package chatagent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionEventHubPublishReleasesLock(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
	}{
		{name: "confirm with full slow subscriber", eventType: EventTypeConfirm},
		{name: "done with full slow subscriber", eventType: EventTypeDone},
		{name: "usage with full slow subscriber", eventType: EventTypeUsage},
		{name: "mode_change with full slow subscriber", eventType: EventTypeModeChange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hub := &SessionEventHub{subs: make(map[string]*ChannelPublisher)}
			slow := hub.Subscribe("slow", 1)
			fast := hub.Subscribe("fast", 8)
			require.NoError(t, slow.Publish(StreamEvent{Type: EventTypeDelta, Text: "fill"}))

			done := make(chan struct{})
			go func() {
				hub.publish(StreamEvent{Type: tt.eventType, ID: "evt-1"})
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("hub publish must not block on a full subscriber")
			}

			select {
			case ev := <-fast.Events():
				assert.Equal(t, tt.eventType, ev.Type)
			case <-time.After(time.Second):
				t.Fatal("fast subscriber did not receive event")
			}

			// Slow buffer still holds the filler; the new event was dropped for it.
			select {
			case ev := <-slow.Events():
				assert.Equal(t, EventTypeDelta, ev.Type)
			default:
				t.Fatal("slow subscriber should still hold the filler event")
			}
		})
	}
}
