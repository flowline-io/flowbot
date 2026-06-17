package chatagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalStreamEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   StreamEvent
		wantSub string
	}{
		{
			name:    "delta event",
			event:   StreamEvent{Type: EventTypeDelta, Text: "hello"},
			wantSub: `"type":"delta"`,
		},
		{
			name: "tool event with stdout",
			event: StreamEvent{
				Type:   EventTypeTool,
				Name:   "run_terminal",
				Status: "running",
				Stdout: "fetching",
			},
			wantSub: `"stdout":"fetching"`,
		},
		{
			name: "subagent inner tool event",
			event: StreamEvent{
				Type:     EventTypeTool,
				Name:     "web_search",
				Subagent: "general-purpose",
				Status:   "running",
				Stdout:   "searching...",
			},
			wantSub: `"subagent":"general-purpose"`,
		},
		{
			name: "confirm resolved timeout",
			event: StreamEvent{
				Type:     EventTypeConfirmResolved,
				ID:       "c-1",
				Approved: false,
				Reason:   string(ConfirmReasonTimeout),
			},
			wantSub: `"reason":"timeout"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MarshalStreamEvent(tt.event)
			require.NoError(t, err)
			assert.Contains(t, data, tt.wantSub)

			frame, err := FormatSSEData(tt.event)
			require.NoError(t, err)
			assert.Contains(t, frame, "data: ")
			assert.Contains(t, frame, tt.wantSub)
		})
	}
}
