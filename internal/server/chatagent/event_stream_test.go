package chatagent

import (
	"testing"
	"time"

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

func TestHandleAPIStreamEventToolLifecycle(t *testing.T) {
	tests := []struct {
		name      string
		events    []agentevent.Event
		wantTypes []string
	}{
		{
			name: "tool start update and end",
			events: []agentevent.Event{
				{Type: agentevent.TypeToolExecutionStart, ToolCall: msg.ToolCallPart{ID: "t1", Name: "bash"}},
				{Type: agentevent.TypeToolExecutionUpdate, ToolCall: msg.ToolCallPart{ID: "t1", Name: "bash"}, Update: "running"},
				{
					Type:       agentevent.TypeToolExecutionEnd,
					DurationMs: 50,
					ToolCall:   msg.ToolCallPart{ID: "t1", Name: "bash"},
					ToolResult: msg.ToolResultMessage{
						Name: "bash", Parts: []msg.ContentPart{msg.TextPart{Text: "done"}},
					},
				},
			},
			wantTypes: []string{EventTypeTool, EventTypeTool, EventTypeTool},
		},
		{
			name: "subagent tool update",
			events: []agentevent.Event{
				{Type: agentevent.TypeToolExecutionStart, ToolCall: msg.ToolCallPart{ID: "s1", Name: delegateSubagentToolName}},
				{Type: agentevent.TypeToolExecutionUpdate, ToolCall: msg.ToolCallPart{ID: "s1", Name: delegateSubagentToolName}, Update: "step:1"},
			},
			wantTypes: []string{EventTypeTool, EventTypeTool},
		},
		{
			name: "turn end publishes done",
			events: []agentevent.Event{
				{Type: agentevent.TypeTurnEnd, TextDelta: "final"},
			},
			wantTypes: []string{EventTypeTurn},
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
			require.Len(t, pub.events, len(tt.wantTypes))
			for i, want := range tt.wantTypes {
				assert.Equal(t, want, pub.events[i].Type)
			}
		})
	}
}

func TestStartAPIEventStream(t *testing.T) {
	tests := []struct {
		name       string
		publisher  EventPublisher
		events     chan agentevent.Event
		wantMinLen int
	}{
		{name: "nil publisher returns immediately", publisher: nil, events: make(chan agentevent.Event), wantMinLen: 0},
		{name: "nil events channel returns immediately", publisher: &apiEventRecorder{}, events: nil, wantMinLen: 0},
		{
			name:      "publishes streamed delta",
			publisher: &apiEventRecorder{},
			events: func() chan agentevent.Event {
				ch := make(chan agentevent.Event, 2)
				ch <- agentevent.Event{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}}
				ch <- agentevent.Event{Type: agentevent.TypeMessageUpdate, TextDelta: "hi"}
				close(ch)
				return ch
			}(),
			wantMinLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			wait := startAPIEventStream(ctx, tt.events, tt.publisher, time.Millisecond)
			wait()
			if rec, ok := tt.publisher.(*apiEventRecorder); ok && tt.wantMinLen > 0 {
				assert.GreaterOrEqual(t, len(rec.events), tt.wantMinLen)
			}
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
