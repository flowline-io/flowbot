package chatagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const toolStatusTemplate = "Running tool: %s..."

// streamUpdateInterval throttles in-progress Slack chat.update calls.
const streamUpdateInterval = time.Second

// streamCoalescer accumulates assistant deltas without blocking the agent event stream.
type streamCoalescer struct {
	mu          sync.Mutex
	accumulated string
	statusText  string
	dirty       bool
}

// newStreamCoalescer creates a coalescer for one agent run.
func newStreamCoalescer() *streamCoalescer {
	return &streamCoalescer{}
}

// reset clears accumulated assistant text for a new assistant generation turn.
func (c *streamCoalescer) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accumulated = ""
	c.statusText = ""
	c.dirty = false
}

// appendDelta adds one text delta from the current assistant turn.
func (c *streamCoalescer) appendDelta(delta string) {
	if delta == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accumulated += delta
	c.statusText = ""
	c.dirty = true
}

// setToolStatus replaces the visible snapshot while tools execute.
func (c *streamCoalescer) setToolStatus(toolName string) {
	if toolName == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statusText = fmt.Sprintf(toolStatusTemplate, toolName)
	c.dirty = true
}

func (c *streamCoalescer) snapshot() (text string, dirty bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.statusText != "" {
		return c.statusText, c.dirty
	}
	return c.accumulated, c.dirty
}

func (c *streamCoalescer) markClean() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dirty = false
}

// startStreamCoalescer consumes agent events and publishes throttled sink updates.
// The returned wait function blocks until the event stream closes.
func startStreamCoalescer(ctx context.Context, events <-chan agentevent.Event, sink StreamSink, interval time.Duration) func() {
	done := make(chan struct{})
	if sink == nil || events == nil {
		close(done)
		return func() { <-done }
	}
	if interval <= 0 {
		interval = time.Second
	}

	go func() {
		defer close(done)
		coalescer := newStreamCoalescer()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-events:
				if !ok {
					publishSnapshot(ctx, coalescer, sink)
					return
				}
				handleStreamEvent(coalescer, ev)
			case <-ticker.C:
				publishSnapshot(ctx, coalescer, sink)
			}
		}
	}()

	return func() { <-done }
}

func handleStreamEvent(coalescer *streamCoalescer, ev agentevent.Event) {
	switch ev.Type {
	case agentevent.TypeMessageStart:
		if _, ok := ev.Message.(msg.AssistantMessage); ok {
			coalescer.reset()
		}
	case agentevent.TypeMessageUpdate:
		coalescer.appendDelta(ev.TextDelta)
	case agentevent.TypeToolExecutionStart:
		if call, ok := ev.ToolCall.(msg.ToolCallPart); ok {
			coalescer.setToolStatus(call.Name)
		}
	}
}

func publishSnapshot(ctx context.Context, coalescer *streamCoalescer, sink StreamSink) {
	text, dirty := coalescer.snapshot()
	if !dirty || text == "" {
		return
	}
	if err := sink.OnDelta(ctx, text); err != nil {
		flog.Warn("[chat-agent] stream delta update: %v", err)
		return
	}
	coalescer.markClean()
}
