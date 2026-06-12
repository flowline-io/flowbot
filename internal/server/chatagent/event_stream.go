package chatagent

import (
	"context"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

const apiStreamUpdateInterval = 150 * time.Millisecond

// startAPIEventStream consumes agent lifecycle events and publishes Chat Agent SSE payloads.
func startAPIEventStream(ctx context.Context, events <-chan agentevent.Event, publisher EventPublisher, interval time.Duration) func() {
	done := make(chan struct{})
	if publisher == nil || events == nil {
		close(done)
		return func() { <-done }
	}
	if interval <= 0 {
		interval = apiStreamUpdateInterval
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
					publishAPIEvent(ctx, publisher, coalescer)
					return
				}
				handleAPIStreamEvent(publisher, coalescer, ev)
			case <-ticker.C:
				publishAPIEvent(ctx, publisher, coalescer)
			}
		}
	}()

	return func() { <-done }
}

func handleAPIStreamEvent(publisher EventPublisher, coalescer *streamCoalescer, ev agentevent.Event) {
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
			_ = publisher.Publish(StreamEvent{
				Type:   EventTypeTool,
				Name:   call.Name,
				Status: "running",
			})
		}
	case agentevent.TypeToolExecutionUpdate:
		if call, ok := ev.ToolCall.(msg.ToolCallPart); ok && ev.Update != "" {
			_ = publisher.Publish(StreamEvent{
				Type:   EventTypeTool,
				Name:   call.Name,
				Status: "running",
				Stdout: ev.Update,
			})
		}
	}
}

func publishAPIEvent(ctx context.Context, publisher EventPublisher, coalescer *streamCoalescer) {
	if ctx.Err() != nil {
		return
	}
	text, dirty := coalescer.snapshot()
	if !dirty || text == "" {
		return
	}
	_ = publisher.Publish(StreamEvent{Type: EventTypeDelta, Text: text})
	coalescer.markClean()
}

// PublishUsageEvent emits a context usage snapshot to the client.
func PublishUsageEvent(publisher EventPublisher, prompt, completion, total, window int, percent float64) {
	if publisher == nil {
		return
	}
	_ = publisher.Publish(StreamEvent{
		Type:             EventTypeUsage,
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
		ContextPercent:   percent,
		ContextWindow:    window,
	})
}
