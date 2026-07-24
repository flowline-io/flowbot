package chatagent_test

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
)

type captureSSE struct {
	events []chatagent.StreamEvent
}

func (c *captureSSE) WriteEvent(event chatagent.StreamEvent) bool {
	c.events = append(c.events, event)
	return event.Type == chatagent.EventTypeDone ||
		event.Type == chatagent.EventTypeError ||
		event.Type == chatagent.EventTypeCanceled
}

func TestBufioSSEWriter_WriteEvent(t *testing.T) {
	tests := []struct {
		name      string
		event     chatagent.StreamEvent
		wantTerm  bool
		wantFrame string
	}{
		{
			name:      "delta is not terminal",
			event:     chatagent.StreamEvent{Type: chatagent.EventTypeDelta, Text: "hi"},
			wantTerm:  false,
			wantFrame: "data: ",
		},
		{
			name:      "done is terminal",
			event:     chatagent.StreamEvent{Type: chatagent.EventTypeDone, Text: "ok"},
			wantTerm:  true,
			wantFrame: "data: ",
		},
		{
			name:      "error is terminal",
			event:     chatagent.StreamEvent{Type: chatagent.EventTypeError, Message: "fail"},
			wantTerm:  true,
			wantFrame: "data: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &chatagent.BufioSSEWriter{W: bufio.NewWriter(&buf)}
			terminal := w.WriteEvent(tt.event)
			assert.Equal(t, tt.wantTerm, terminal)
			require.NoError(t, bufio.NewWriter(&buf).Flush())
			out := buf.String()
			assert.True(t, strings.HasPrefix(out, tt.wantFrame))
			assert.Contains(t, out, tt.event.Type)
		})
	}
}

func TestDrainPublisherSSE(t *testing.T) {
	tests := []struct {
		name      string
		events    []chatagent.StreamEvent
		wantCount int
		wantTerm  bool
	}{
		{
			name: "drains buffered events",
			events: []chatagent.StreamEvent{
				{Type: chatagent.EventTypeDelta, Text: "a"},
				{Type: chatagent.EventTypeDelta, Text: "b"},
			},
			wantCount: 2,
		},
		{
			name:      "empty publisher is no-op",
			events:    nil,
			wantCount: 0,
		},
		{
			name: "terminal event stops drain",
			events: []chatagent.StreamEvent{
				{Type: chatagent.EventTypeDone, Text: "ok"},
			},
			wantCount: 1,
			wantTerm:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := chatagent.NewChannelPublisher(8)
			for _, ev := range tt.events {
				require.NoError(t, pub.Publish(ev))
			}
			captured := &captureSSE{}
			chatagent.DrainPublisherSSE(captured, pub)
			assert.Len(t, captured.events, tt.wantCount)
			if tt.wantTerm {
				assert.Equal(t, chatagent.EventTypeDone, captured.events[len(captured.events)-1].Type)
			}
		})
	}
}

func TestStreamAPIRun_InFlight(t *testing.T) {
	svc := chatagent.NewService()
	sessionID := "sess-inflight"
	pub := chatagent.NewChannelPublisher(4)
	gate := chatagent.NewConfirmGate(sessionID, pub, nil)
	require.NoError(t, svc.TrySetAPIRunState(sessionID, chatagent.NewAPIRunState(pub, gate)))
	t.Cleanup(func() {
		svc.ClearAPIRunState(sessionID, nil)
	})

	captured := &captureSSE{}
	svc.StreamAPIRun(context.Background(), sessionID, "hello", nil, "", captured)

	require.Len(t, captured.events, 1)
	assert.Equal(t, chatagent.EventTypeError, captured.events[0].Type)
	assert.Contains(t, captured.events[0].Message, "run already in progress")
}
