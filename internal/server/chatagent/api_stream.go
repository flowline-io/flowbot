package chatagent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"sync"

	fbtrace "github.com/flowline-io/flowbot/pkg/trace"
)

// SSEWriter writes chat agent stream events to an HTTP response body.
type SSEWriter interface {
	WriteEvent(event StreamEvent) (terminal bool)
}

// BufioSSEWriter writes SSE frames to a buffered writer.
type BufioSSEWriter struct {
	W *bufio.Writer
}

// WriteEvent serializes one event and flushes it to the stream.
// Returns true when the stream should stop: I/O failure, encode failure, or a
// terminal event type was written successfully.
func (b *BufioSSEWriter) WriteEvent(event StreamEvent) bool {
	frame, err := FormatSSEData(event)
	if err != nil {
		return true
	}
	if _, err := fmt.Fprint(b.W, frame); err != nil {
		return true
	}
	if err := b.W.Flush(); err != nil {
		return true
	}
	return isTerminalStreamEvent(event)
}

func isTerminalStreamEvent(event StreamEvent) bool {
	return event.Type == EventTypeDone ||
		event.Type == EventTypeError ||
		event.Type == EventTypeCanceled
}

// DrainPublisherSSE drains buffered publisher events to w until empty or terminal.
func DrainPublisherSSE(w SSEWriter, publisher *ChannelPublisher) {
	for {
		select {
		case ev, ok := <-publisher.Events():
			if !ok {
				return
			}
			if w.WriteEvent(ev) {
				return
			}
		default:
			return
		}
	}
}

// StreamAPIRun executes one agent turn and streams SSE events to w.
func StreamAPIRun(ctx context.Context, svc *Service, sessionID, text string, attachments []AttachmentRef, ownerUID string, w SSEWriter) {
	hub := GetSessionEventHub(sessionID)
	subID := "run"
	publisher := hub.Subscribe(subID, 64)
	var detachOnce sync.Once
	detachFromHub := func() {
		detachOnce.Do(func() {
			// Remove from fan-out without Close: RunAPI still owns this publisher
			// until the turn finishes. Closing here used to drop Done after a
			// mid-turn SSE write failure, so the browser saw a clean stream end
			// with no assistant content and no reload trigger.
			hub.Detach(subID)
		})
	}
	defer detachFromHub()

	gate := NewConfirmGate(sessionID, nil)
	runState := NewAPIRunState(publisher, gate)
	if err := TrySetAPIRunState(sessionID, runState); err != nil {
		publisher.Close()
		_ = w.WriteEvent(StreamEvent{
			Type:    EventTypeError,
			Message: err.Error(),
		})
		return
	}

	// Detach from the HTTP request context so closing the message SSE stream
	// does not cancel a run that is still waiting for tool approval.
	runCtx, cancel := fbtrace.DetachWithTimeout(ctx, RunTimeout())
	BindRunCancel(sessionID, cancel)

	runDone := make(chan error, 1)
	go func() {
		var runErr error
		defer func() {
			cancel()
			UnbindRunCancel(sessionID)
			ClearAPIRunState(sessionID, runState)
			// Observers (reopened detail pages) do not share the primary messages
			// SSE; tell them history is ready to reload.
			PublishSessionEvent(sessionID, StreamEvent{Type: EventTypeRunComplete})
			runDone <- runErr
		}()
		runErr = svc.RunAPI(runCtx, RunRequest{
			SessionID:   sessionID,
			Text:        text,
			Attachments: attachments,
		}, &APIRunOptions{
			Publisher: publisher,
			Confirm:   gate,
			OwnerUID:  ownerUID,
		})
		publisher.Close()
	}()

	// When the browser disconnects, detach from the hub immediately so confirm /
	// run_complete fan-out is not blocked on this abandoned subscriber.
	writeStreamEventsUntilRunDone(w, publisher, runDone, detachFromHub)
}

// writeStreamEventsUntilRunDone forwards publisher events to w until the publisher closes,
// then waits for runDone. Callers must Close the publisher before sending on runDone so this
// loop cannot block forever draining an open channel. Waiting until both complete avoids racing
// HTTP/test cleanup with post-Done work (title generation, ClearAPIRunState) and ensures errors
// are still written when the publisher closes with no events (e.g. empty message).
// onDetach is invoked once the SSE writer fails or ends, so hub fan-out can drop this consumer.
func writeStreamEventsUntilRunDone(w SSEWriter, publisher *ChannelPublisher, runDone <-chan error, onDetach func()) {
	stopWriting := false
	detached := false
	detach := func() {
		if detached || onDetach == nil {
			return
		}
		detached = true
		onDetach()
	}

	for {
		ev, ok := <-publisher.Events()
		if !ok {
			break
		}
		if stopWriting {
			continue
		}
		if !w.WriteEvent(ev) {
			continue
		}
		// WriteEvent stopped us: either the terminal event was delivered, or I/O failed.
		detach()
		if isTerminalStreamEvent(ev) {
			stopWriting = true
			continue
		}
		// I/O failure on a non-terminal event: keep the publisher open and still
		// attempt to deliver a later Done/Error/Canceled to a half-open stream.
		continue
	}

	runErr := <-runDone
	if stopWriting || runErr == nil {
		return
	}
	if errors.Is(runErr, context.Canceled) {
		_ = w.WriteEvent(StreamEvent{
			Type:    EventTypeCanceled,
			Message: "run canceled by user",
		})
		return
	}
	_ = w.WriteEvent(StreamEvent{
		Type:    EventTypeError,
		Message: runErr.Error(),
	})
}
