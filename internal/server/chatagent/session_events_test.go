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
		{name: "confirm while subscriber blocked", eventType: EventTypeConfirm},
		{name: "done while subscriber blocked", eventType: EventTypeDone},
		{name: "usage while subscriber blocked", eventType: EventTypeUsage},
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
				t.Fatal("publish should not block while holding hub lock")
			case <-time.After(50 * time.Millisecond):
			}

			<-slow.Events()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("publish did not complete after slow subscriber consumed")
			}

			select {
			case ev := <-fast.Events():
				assert.Equal(t, tt.eventType, ev.Type)
			case <-time.After(time.Second):
				t.Fatal("fast subscriber did not receive event")
			}
		})
	}
}
