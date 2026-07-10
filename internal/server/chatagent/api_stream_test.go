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

func TestStreamAPIRun_InFlight(t *testing.T) {
	sessionID := "sess-inflight"
	pub := chatagent.NewChannelPublisher(4)
	gate := chatagent.NewConfirmGate(sessionID, pub)
	require.NoError(t, chatagent.TrySetAPIRunState(sessionID, chatagent.NewAPIRunState(pub, gate)))
	t.Cleanup(func() {
		chatagent.ClearAPIRunState(sessionID, nil)
	})

	captured := &captureSSE{}
	chatagent.StreamAPIRun(context.Background(), chatagent.NewService(), sessionID, "hello", captured)

	require.Len(t, captured.events, 1)
	assert.Equal(t, chatagent.EventTypeError, captured.events[0].Type)
	assert.Contains(t, captured.events[0].Message, "run already in progress")
}
