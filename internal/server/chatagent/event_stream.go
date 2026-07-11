package chatagent

import (
	"context"
	"fmt"
	"strings"
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
	reasoningStarted   bool
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
		handleAPIMessageStart(tracker, ev)
	case agentevent.TypeMessageUpdate:
		handleAPIMessageUpdate(tracker, ev)
	case agentevent.TypeMessageEnd:
		publishAPIMessageEnd(publisher, tracker, ev)
	case agentevent.TypeToolExecutionStart:
		publishAPIToolStart(publisher, tracker, ev)
	case agentevent.TypeToolExecutionUpdate:
		publishAPIToolUpdate(publisher, tracker, ev)
	case agentevent.TypeToolExecutionEnd:
		publishAPIToolEnd(publisher, tracker, ev)
	case agentevent.TypeTurnEnd:
		publishAPITurnEnd(publisher, ev)
	}
}

func handleAPIMessageStart(tracker *apiStreamTracker, ev agentevent.Event) {
	if _, ok := ev.Message.(msg.AssistantMessage); !ok {
		return
	}
	tracker.coalescer.reset()
	tracker.reasoningCoalescer.reset()
	tracker.subagentTool = ""
	tracker.reasoningStarted = false
}

func handleAPIMessageUpdate(tracker *apiStreamTracker, ev agentevent.Event) {
	if ev.ReasoningDelta != "" {
		tracker.reasoningStarted = true
		tracker.reasoningCoalescer.appendDelta(ev.ReasoningDelta)
	}
	if ev.TextDelta != "" {
		tracker.coalescer.appendDelta(ev.TextDelta)
	}
}

func publishAPIMessageEnd(publisher EventPublisher, tracker *apiStreamTracker, ev agentevent.Event) {
	assistant, ok := ev.Message.(msg.AssistantMessage)
	if !ok || !tracker.reasoningStarted {
		return
	}
	_ = publisher.Publish(StreamEvent{
		Type:       EventTypeThinking,
		Status:     "completed",
		DurationMs: assistant.ThinkingDurationMs,
	})
	tracker.reasoningStarted = false
}

func publishAPIToolStart(publisher EventPublisher, tracker *apiStreamTracker, ev agentevent.Event) {
	call, ok := ev.ToolCall.(msg.ToolCallPart)
	if !ok {
		return
	}
	tracker.coalescer.setToolStatus(toolStatusText(call))
	if call.Name == taskToolName {
		tracker.subagentTool = ""
		_ = publisher.Publish(taskToolStreamEvent(call, "running", "", 0))
		return
	}
	tracker.subagentTool = ""
	_ = publisher.Publish(StreamEvent{
		Type:   EventTypeTool,
		Name:   call.Name,
		Status: "running",
	})
}

func publishAPIToolUpdate(publisher EventPublisher, tracker *apiStreamTracker, ev agentevent.Event) {
	call, ok := ev.ToolCall.(msg.ToolCallPart)
	if !ok || ev.Update == "" {
		return
	}
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

func publishAPIToolEnd(publisher EventPublisher, tracker *apiStreamTracker, ev agentevent.Event) {
	call, ok := ev.ToolCall.(msg.ToolCallPart)
	if !ok {
		return
	}
	result, ok := ev.ToolResult.(msg.ToolResultMessage)
	if !ok {
		return
	}
	status := "completed"
	if result.IsError {
		status = "error"
	}
	stdout := apiToolResultText(result)

	if call.Name == taskToolName {
		tracker.subagentTool = ""
		_ = publisher.Publish(taskToolStreamEvent(call, status, stdout, ev.DurationMs))
		return
	}

	_ = publisher.Publish(StreamEvent{
		Type:       EventTypeTool,
		Name:       toolDisplayName(call),
		Status:     status,
		Stdout:     stdout,
		DurationMs: ev.DurationMs,
	})
}

func publishAPITurnEnd(publisher EventPublisher, ev agentevent.Event) {
	_ = publisher.Publish(StreamEvent{
		Type:       EventTypeTurn,
		DurationMs: ev.DurationMs,
		Step:       ev.Step,
	})
}

func apiToolResultText(result msg.ToolResultMessage) string {
	var text strings.Builder
	for _, part := range result.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = text.WriteString(tp.Text)
		}
	}
	return text.String()
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
		_ = publisher.Publish(subagentInnerToolStreamEvent(subagent, activeTool, "running", detail, 0))
		return
	}
	if activeTool != "" {
		_ = publisher.Publish(subagentInnerToolStreamEvent(subagent, activeTool, "running", "", 0))
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
