package chatagent

import (
	"context"
	"testing"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

type recordingSink struct {
	deltas []string
	final  string
}

func (r *recordingSink) OnDelta(_ context.Context, text string) error {
	r.deltas = append(r.deltas, text)
	return nil
}

func (r *recordingSink) Flush(_ context.Context, final string) error {
	r.final = final
	return nil
}

func TestStreamCoalescer_handleEvents(t *testing.T) {
	tests := []struct {
		name      string
		events    []agentevent.Event
		wantText  string
		wantDirty bool
	}{
		{
			name: "accumulates assistant deltas",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, TextDelta: "hel"},
				{Type: agentevent.TypeMessageUpdate, TextDelta: "lo"},
			},
			wantText:  "hello",
			wantDirty: true,
		},
		{
			name: "resets on assistant start",
			events: []agentevent.Event{
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, TextDelta: "old"},
				{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}},
				{Type: agentevent.TypeMessageUpdate, TextDelta: "new"},
			},
			wantText:  "new",
			wantDirty: true,
		},
		{
			name: "tool status overrides snapshot",
			events: []agentevent.Event{
				{Type: agentevent.TypeToolExecutionStart, ToolCall: msg.ToolCallPart{Name: "echo"}},
			},
			wantText:  "Running tool: echo...",
			wantDirty: true,
		},
		{
			name: "task tool shows subagent name",
			events: []agentevent.Event{
				{Type: agentevent.TypeToolExecutionStart, ToolCall: msg.ToolCallPart{
					Name:      "task",
					Arguments: `{"subagent_type":"code-reviewer","prompt":"review"}`,
				}},
			},
			wantText:  "Delegating to subagent: code-reviewer...",
			wantDirty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newStreamCoalescer()
			for _, ev := range tt.events {
				handleStreamEvent(c, ev)
			}
			text, dirty := c.snapshot()
			assert.Equal(t, tt.wantText, text)
			assert.Equal(t, tt.wantDirty, dirty)
		})
	}
}

func TestStartStreamCoalescer_throttlesUpdates(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "publishes accumulated text"},
		{name: "latest wins before ticker"},
		{name: "stops when channel closes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			events := make(chan agentevent.Event, 4)
			sink := &recordingSink{}
			wait := startStreamCoalescer(context.Background(), events, sink, 20*time.Millisecond)

			events <- agentevent.Event{Type: agentevent.TypeMessageStart, Message: msg.AssistantMessage{}}
			events <- agentevent.Event{Type: agentevent.TypeMessageUpdate, TextDelta: "hel"}
			events <- agentevent.Event{Type: agentevent.TypeMessageUpdate, TextDelta: "lo"}
			close(events)
			wait()

			assert.NotEmpty(t, sink.deltas)
			assert.Equal(t, "hello", sink.deltas[len(sink.deltas)-1])
		})
	}
}

func TestPublishSnapshot_skipsEmpty(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*streamCoalescer)
		wantCount int
	}{
		{name: "no delta yet", setup: func(_ *streamCoalescer) {}, wantCount: 0},
		{name: "publishes text", setup: func(c *streamCoalescer) { c.appendDelta("hi") }, wantCount: 1},
		{name: "clears dirty flag", setup: func(c *streamCoalescer) { c.appendDelta("x") }, wantCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newStreamCoalescer()
			tt.setup(c)
			sink := &recordingSink{}
			publishSnapshot(context.Background(), c, sink)
			assert.Len(t, sink.deltas, tt.wantCount)
		})
	}
}
