package chatagent

import (
	"context"
	"fmt"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

const apiStreamUpdateInterval = 150 * time.Millisecond

// apiStreamTracker tracks subagent inner-tool progress for one API SSE connection.
type apiStreamTracker struct {
	coalescer          *streamCoalescer
	reasoningCoalescer *streamCoalescer
	subagentTool       string
}

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
		tracker := &apiStreamTracker{
			coalescer:          newStreamCoalescer(),
			reasoningCoalescer: newStreamCoalescer(),
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-events:
				if !ok {
					publishAPIEvent(ctx, publisher, tracker.coalescer)
					publishAPIReasoningEvent(ctx, publisher, tracker.reasoningCoalescer)
					return
				}
				handleAPIStreamEvent(publisher, tracker, ev)
			case <-ticker.C:
				publishAPIEvent(ctx, publisher, tracker.coalescer)
				publishAPIReasoningEvent(ctx, publisher, tracker.reasoningCoalescer)
			}
		}
	}()

	return func() { <-done }
}

func handleAPIStreamEvent(publisher EventPublisher, tracker *apiStreamTracker, ev agentevent.Event) {
	switch ev.Type {
	case agentevent.TypeMessageStart:
		if _, ok := ev.Message.(msg.AssistantMessage); ok {
			tracker.coalescer.reset()
			tracker.reasoningCoalescer.reset()
			tracker.subagentTool = ""
		}
	case agentevent.TypeMessageUpdate:
		if ev.ReasoningDelta != "" {
			tracker.reasoningCoalescer.appendDelta(ev.ReasoningDelta)
		}
		if ev.TextDelta != "" {
			tracker.coalescer.appendDelta(ev.TextDelta)
		}
	case agentevent.TypeToolExecutionStart:
		if call, ok := ev.ToolCall.(msg.ToolCallPart); ok {
			tracker.coalescer.setToolStatus(toolStatusText(call))
			if call.Name == taskToolName {
				tracker.subagentTool = ""
				_ = publisher.Publish(taskToolStreamEvent(call, "running", ""))
				return
			}
			tracker.subagentTool = ""
			_ = publisher.Publish(StreamEvent{
				Type:   EventTypeTool,
				Name:   call.Name,
				Status: "running",
			})
		}
	case agentevent.TypeToolExecutionUpdate:
		if call, ok := ev.ToolCall.(msg.ToolCallPart); ok && ev.Update != "" {
			if call.Name == taskToolName {
				publishSubagentToolUpdate(publisher, tracker, ev.Update)
				return
			}
			_ = publisher.Publish(StreamEvent{
				Type:   EventTypeTool,
				Name:   toolDisplayName(call),
				Status: "running",
				Stdout: ev.Update,
			})
		}
	}
}

func publishSubagentToolUpdate(publisher EventPublisher, tracker *apiStreamTracker, update string) {
	subagent, tool, detail, ok := parseSubagentProgress(update)
	if !ok {
		_ = publisher.Publish(StreamEvent{
			Type:   EventTypeTool,
			Name:   taskToolName,
			Status: "running",
			Stdout: update,
		})
		return
	}
	if tool != "" {
		tracker.subagentTool = tool
	}
	activeTool := tracker.subagentTool
	tracker.coalescer.setToolStatus(subagentToolStatusText(subagent, activeTool, detail))
	if detail != "" {
		_ = publisher.Publish(subagentInnerToolStreamEvent(subagent, activeTool, "running", detail))
		return
	}
	if activeTool != "" {
		_ = publisher.Publish(subagentInnerToolStreamEvent(subagent, activeTool, "running", ""))
	}
}

// toolDisplayName returns the client-facing tool name, annotating the task tool
// with the delegated subagent so CLI and web clients can show the delegation target.
func toolDisplayName(call msg.ToolCallPart) string {
	if call.Name == taskToolName {
		if name := subagentTypeFromArgs(call.Arguments); name != "" {
			return fmt.Sprintf("%s (%s)", taskToolName, name)
		}
	}
	return call.Name
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

func publishAPIReasoningEvent(ctx context.Context, publisher EventPublisher, coalescer *streamCoalescer) {
	if ctx.Err() != nil {
		return
	}
	text, dirty := coalescer.snapshot()
	if !dirty || text == "" {
		return
	}
	_ = publisher.Publish(StreamEvent{Type: EventTypeThinking, Text: text})
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
