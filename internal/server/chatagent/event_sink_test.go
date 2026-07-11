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
		{name: "blocks mode_change when full", eventType: EventTypeModeChange, critical: true},
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

func TestChannelPublisherTimingEventsCritical(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		event    StreamEvent
		critical bool
	}{
		{
			name:     "turn blocks when full",
			event:    StreamEvent{Type: EventTypeTurn, DurationMs: 900, Step: 1},
			critical: true,
		},
		{
			name:     "tool completed blocks when full",
			event:    StreamEvent{Type: EventTypeTool, Name: "echo", Status: "completed", DurationMs: 120},
			critical: true,
		},
		{
			name:     "thinking completed blocks when full",
			event:    StreamEvent{Type: EventTypeThinking, Status: "completed", DurationMs: 300},
			critical: true,
		},
		{
			name:     "tool running drops when full",
			event:    StreamEvent{Type: EventTypeTool, Name: "echo", Status: "running"},
			critical: false,
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

			if tt.critical {
				select {
				case <-done:
					t.Fatal("timing publish should block until buffer is consumed")
				case <-time.After(50 * time.Millisecond):
				}
				<-pub.Events()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("timing publish did not complete after consume")
				}
				ev := <-pub.Events()
				assert.Equal(t, tt.event.Type, ev.Type)
				return
			}

			select {
			case <-done:
			case <-time.After(50 * time.Millisecond):
				t.Fatal("running tool publish should return immediately when full")
			}
		})
	}
}
