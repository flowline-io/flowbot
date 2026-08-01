package chatagent

import (
	"sync"
	"testing"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiEventRecorder struct {
	mu     sync.Mutex
	events []StreamEvent
}

func (r *apiEventRecorder) Publish(event StreamEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}

func (r *apiEventRecorder) snapshot() []StreamEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]StreamEvent, len(r.events))
	copy(out, r.events)
	return out
}

func TestHandleAPIStreamEventReasoning(t *testing.T) {
	tests := []struct {
		name       string
		events     []agentevent.Event
		wantEvents []StreamEvent
	}{
		{
			name: "first reasoning delta flushes immediately then accumulates",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "plan"},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "ning"},
			},
			wantEvents: []StreamEvent{
				{Type: EventTypeThinking, Text: "plan"},
				{Type: EventTypeThinking, Text: "planning"},
			},
		},
		{
			name: "keeps reasoning separate from answer delta",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "think"},
				{Type: agentevent.TypeMessageUpdate, TextDelta: "hello"},
			},
			wantEvents: []StreamEvent{
				{Type: EventTypeThinking, Text: "think"},
				{Type: EventTypeDelta, Text: "hello"},
			},
		},
		{
			name: "resets reasoning on assistant start",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "old"},
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "new"},
			},
			wantEvents: []StreamEvent{
				{Type: EventTypeThinking, Text: "old"},
				{Type: EventTypeThinking, Text: "new"},
			},
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
				handleAPIStreamEvent(t.Context(), pub, tracker, ev)
			}
			publishAPIEvent(t.Context(), pub, tracker.coalescer)
			publishAPIReasoningEvent(t.Context(), pub, tracker.reasoningCoalescer)

			require.Equal(t, tt.wantEvents, pub.snapshot())
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
				handleAPIStreamEvent(t.Context(), pub, tracker, ev)
			}
			got := pub.snapshot()
			require.Len(t, got, len(tt.wantTypes))
			for i, want := range tt.wantTypes {
				assert.Equal(t, want, got[i].Type)
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
			if _, ok := tt.publisher.(*apiEventRecorder); ok && tt.wantMinLen > 0 {
				assert.GreaterOrEqual(t, len(tt.publisher.(*apiEventRecorder).snapshot()), tt.wantMinLen)
			}
		})
	}
}

func TestStartAPIEventStreamFirstDeltaImmediate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		updates    []agentevent.Event
		wantFirst  StreamEvent
		wantFinal  StreamEvent
		wantCount  int
	}{
		{
			name: "text first flush then coalesce on close",
			updates: []agentevent.Event{
				{Type: agentevent.TypeMessageUpdate, TextDelta: "hi"},
				{Type: agentevent.TypeMessageUpdate, TextDelta: " there"},
			},
			wantFirst: StreamEvent{Type: EventTypeDelta, Text: "hi"},
			wantFinal: StreamEvent{Type: EventTypeDelta, Text: "hi there"},
			wantCount: 2,
		},
		{
			name: "thinking first flush then coalesce on close",
			updates: []agentevent.Event{
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "plan"},
				{Type: agentevent.TypeMessageUpdate, ReasoningDelta: "ning"},
			},
			wantFirst: StreamEvent{Type: EventTypeThinking, Text: "plan"},
			wantFinal: StreamEvent{Type: EventTypeThinking, Text: "planning"},
			wantCount: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := &apiEventRecorder{}
			events := make(chan agentevent.Event, 4)
			wait := startAPIEventStream(t.Context(), events, pub, time.Hour)

			events <- agentevent.Event{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}}
			events <- tt.updates[0]
			require.Eventually(t, func() bool {
				return len(pub.snapshot()) >= 1
			}, time.Second, 5*time.Millisecond)
			assert.Equal(t, tt.wantFirst, pub.snapshot()[0])

			events <- tt.updates[1]
			close(events)
			wait()
			got := pub.snapshot()
			require.Len(t, got, tt.wantCount)
			assert.Equal(t, tt.wantFinal, got[len(got)-1])
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
			wantCount: 2,
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
				handleAPIStreamEvent(t.Context(), pub, tracker, ev)
			}
			got := pub.snapshot()
			require.Len(t, got, tt.wantCount)
			assert.Equal(t, tt.wantLast, got[len(got)-1])
		})
	}
}
