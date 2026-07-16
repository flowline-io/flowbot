package chatagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsObserverStreamEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      bool
	}{
		{name: "confirm is observer", eventType: EventTypeConfirm, want: true},
		{name: "confirm_resolved is observer", eventType: EventTypeConfirmResolved, want: true},
		{name: "canceled is observer", eventType: EventTypeCanceled, want: true},
		{name: "mode_change is observer", eventType: EventTypeModeChange, want: true},
		{name: "delta is not observer", eventType: EventTypeDelta, want: false},
		{name: "done is not observer", eventType: EventTypeDone, want: false},
		{name: "tool is not observer", eventType: EventTypeTool, want: false},
		{name: "empty type is not observer", eventType: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsObserverStreamEvent(tt.eventType))
		})
	}
}

func TestMarshalStreamEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   StreamEvent
		wantSub string
	}{
		{
			name:    "thinking event",
			event:   StreamEvent{Type: EventTypeThinking, Text: "planning"},
			wantSub: `"type":"thinking"`,
		},
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
