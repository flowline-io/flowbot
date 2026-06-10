package event_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStream_PushAndEnd(t *testing.T) {
	tests := []struct {
		name      string
		buffer    int
		eventType event.Type
	}{
		{name: "agent start event", buffer: 8, eventType: event.TypeAgentStart},
		{name: "turn end event", buffer: 4, eventType: event.TypeTurnEnd},
		{name: "default buffer", buffer: 0, eventType: event.TypeAgentEnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stream := event.NewStream(tt.buffer)
			called := false
			stream.Subscribe(func(ev event.Event) error {
				called = true
				assert.Equal(t, tt.eventType, ev.Type)
				return nil
			})

			ctx := context.Background()
			require.NoError(t, stream.Push(ctx, event.Event{Type: tt.eventType}))
			stream.End(nil, nil)

			result, err := stream.Await(ctx)
			require.NoError(t, err)
			assert.True(t, called)
			assert.Nil(t, result.Err)
		})
	}
}

func TestStream_AwaitCancelledContext(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cancel before end"},
		{name: "cancel waiting result"},
		{name: "cancel after subscribe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stream := event.NewStream(4)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := stream.Await(ctx)
			assert.Error(t, err)
		})
	}
}

func TestStream_SubscribeNilHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "nil handler ignored"},
		{name: "second nil ignored"},
		{name: "valid handler still runs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stream := event.NewStream(2)
			stream.Subscribe(nil)
			called := false
			stream.Subscribe(func(_ event.Event) error {
				called = true
				return nil
			})
			require.NoError(t, stream.Push(context.Background(), event.Event{Type: event.TypeTurnStart}))
			stream.End(nil, nil)
			_, err := stream.Await(context.Background())
			require.NoError(t, err)
			assert.True(t, called)
		})
	}
}
