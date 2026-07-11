package chatagent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
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
func StreamAPIRun(ctx context.Context, svc *Service, sessionID, text string, w SSEWriter) {
	hub := GetSessionEventHub(sessionID)
	subID := "run"
	publisher := hub.Subscribe(subID, 64)
	defer hub.Unsubscribe(subID)

	gate := NewConfirmGate(sessionID, nil)
	runState := NewAPIRunState(publisher, gate)
	if err := TrySetAPIRunState(sessionID, runState); err != nil {
		_ = w.WriteEvent(StreamEvent{
			Type:    EventTypeError,
			Message: err.Error(),
		})
		return
	}

	// Detach from the HTTP request context so closing the message SSE stream
	// does not cancel a run that is still waiting for tool approval.
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), RunTimeout())
	BindRunCancel(sessionID, cancel)

	runDone := make(chan error, 1)
	go func() {
		defer func() {
			cancel()
			UnbindRunCancel(sessionID)
			ClearAPIRunState(sessionID, runState)
		}()
		runDone <- svc.RunAPI(runCtx, RunRequest{
			SessionID: sessionID,
			Text:      text,
		}, &APIRunOptions{
			Publisher: publisher,
			Confirm:   gate,
		})
		publisher.Close()
	}()

	for {
		select {
		case ev, ok := <-publisher.Events():
			if !ok {
				return
			}
			if w.WriteEvent(ev) {
				return
			}
		case runErr := <-runDone:
			for {
				ev, ok := <-publisher.Events()
				if !ok {
					break
				}
				if w.WriteEvent(ev) {
					return
				}
			}
			if runErr != nil {
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
			return
		}
	}
}
