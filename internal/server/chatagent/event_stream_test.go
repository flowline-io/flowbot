package chatagent

import (
	"testing"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiEventRecorder struct {
	events []StreamEvent
}

func (r *apiEventRecorder) Publish(event StreamEvent) error {
	r.events = append(r.events, event)
	return nil
}

func TestHandleAPIStreamEventReasoning(t *testing.T) {
	tests := []struct {
		name      string
		events    []agentevent.Event
		wantType  string
		wantText  string
		wantCount int
	}{
		{
			name: "accumulates reasoning deltas",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "plan"},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "ning"},
			},
			wantType:  EventTypeThinking,
			wantText:  "planning",
			wantCount: 1,
		},
		{
			name: "keeps reasoning separate from answer delta",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "think"},
				{Type: agentevent.TypeMessageUpdate, TextDelta: "hello"},
			},
			wantType:  EventTypeThinking,
			wantText:  "think",
			wantCount: 2,
		},
		{
			name: "resets reasoning on assistant start",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "old"},
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "new"},
			},
			wantType:  EventTypeThinking,
			wantText:  "new",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := &apiEventRecorder{}
			tracker := &apiStreamTracker{
				coalescer:          newStreamCoalescer(),
				reasoningCoalescer: newStreamCoalescer(),
			}
			for _, ev := range tt.events {
				handleAPIStreamEvent(pub, tracker, ev)
			}
			publishAPIEvent(t.Context(), pub, tracker.coalescer)
			publishAPIReasoningEvent(t.Context(), pub, tracker.reasoningCoalescer)

			require.Len(t, pub.events, tt.wantCount)
			last := pub.events[len(pub.events)-1]
			assert.Equal(t, tt.wantType, last.Type)
			assert.Equal(t, tt.wantText, last.Text)
		})
	}
}

func TestHandleAPIStreamEventTiming(t *testing.T) {
	tests := []struct {
		name      string
		events    []agentevent.Event
		wantLast  StreamEvent
		wantCount int
	}{
		{
			name: "tool end publishes completed duration",
			events: []agentevent.Event{
				{
					Type:       agentevent.TypeToolExecutionEnd,
					DurationMs: 120,
					ToolCall:   msg.ToolCallPart{ID: "1", Name: "echo"},
					ToolResult: msg.ToolResultMessage{
						Name:  "echo",
						Parts: []msg.ContentPart{msg.TextPart{Text: "ok"}},
					},
				},
			},
			wantLast: StreamEvent{
				Type:       EventTypeTool,
				Name:       "echo",
				Status:     "completed",
				Stdout:     "ok",
				DurationMs: 120,
			},
			wantCount: 1,
		},
		{
			name: "thinking completed on message end",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "plan"},
				{Type: agentevent.TypeMessageEnd, Message: msg.AssistantMessage{ThinkingDurationMs: 450}},
			},
			wantLast: StreamEvent{
				Type:       EventTypeThinking,
				Status:     "completed",
				DurationMs: 450,
			},
			wantCount: 1,
		},
		{
			name: "turn end publishes step duration",
			events: []agentevent.Event{
				{Type: agentevent.TypeTurnEnd, DurationMs: 1800, Step: 2},
			},
			wantLast: StreamEvent{
				Type:       EventTypeTurn,
				DurationMs: 1800,
				Step:       2,
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := &apiEventRecorder{}
			tracker := &apiStreamTracker{
				coalescer:          newStreamCoalescer(),
				reasoningCoalescer: newStreamCoalescer(),
			}
			for _, ev := range tt.events {
				handleAPIStreamEvent(pub, tracker, ev)
			}
			require.Len(t, pub.events, tt.wantCount)
			assert.Equal(t, tt.wantLast, pub.events[len(pub.events)-1])
		})
	}
}
