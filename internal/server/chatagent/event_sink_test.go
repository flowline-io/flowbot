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
		critical  bool
	}{
		{name: "drops delta when full", eventType: EventTypeDelta, critical: false},
		{name: "blocks confirm when full", eventType: EventTypeConfirm, critical: true},
		{name: "blocks done when full", eventType: EventTypeDone, critical: true},
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

			if tt.critical {
				select {
				case <-done:
					t.Fatal("critical publish should block until buffer is consumed")
				case <-time.After(50 * time.Millisecond):
				}
				<-pub.Events()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("critical publish did not complete after consume")
				}
				ev := <-pub.Events()
				assert.Equal(t, tt.eventType, ev.Type)
				return
			}

			select {
			case <-done:
			case <-time.After(50 * time.Millisecond):
				t.Fatal("delta publish should return immediately when full")
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
