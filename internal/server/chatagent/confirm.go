package chatagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/google/uuid"
)

const defaultConfirmTimeout = 5 * time.Minute

// ConfirmReason describes how a tool confirmation was resolved.
type ConfirmReason string

const (
	ConfirmReasonApproved ConfirmReason = "approved"
	ConfirmReasonDenied   ConfirmReason = "denied"
	ConfirmReasonTimeout  ConfirmReason = "timeout"
)

// ConfirmResponse is the user's answer to a pending tool confirmation.
type ConfirmResponse struct {
	Approved bool
	Reason   ConfirmReason
}

// ConfirmGate blocks dangerous tool execution until the client approves or denies.
type ConfirmGate struct {
	mu        sync.Mutex
	id        string
	sessionID string
	publisher EventPublisher
	ch        chan ConfirmResponse
	done      chan struct{}
	resolved  bool
	timeout   time.Duration
}

// NewConfirmGate creates a gate that publishes confirm events to the active SSE stream.
func NewConfirmGate(sessionID string, publisher EventPublisher) *ConfirmGate {
	return &ConfirmGate{
		id:        uuid.NewString(),
		sessionID: sessionID,
		publisher: publisher,
		ch:        make(chan ConfirmResponse, 1),
		done:      make(chan struct{}),
		timeout:   defaultConfirmTimeout,
	}
}

// ID returns the confirmation request identifier shared with the client.
func (g *ConfirmGate) ID() string {
	return g.id
}

// Wait publishes a confirm event and blocks until the client responds or times out.
func (g *ConfirmGate) Wait(ctx context.Context, event hooks.ToolCallEvent) (bool, error) {
	confirmID := g.beginWait()
	summary := formatConfirmSummary(event)
	if g.publisher != nil {
		_ = g.publisher.Publish(StreamEvent{
			Type:    EventTypeConfirm,
			ID:      confirmID,
			Tool:    event.ToolCall.Name,
			Summary: summary,
		})
	}

	timer := time.NewTimer(g.timeout)
	defer timer.Stop()

	select {
	case resp := <-g.ch:
		g.publishResolved(confirmID, resp)
		return resp.Approved, nil
	case <-timer.C:
		g.publishResolved(confirmID, ConfirmResponse{Approved: false, Reason: ConfirmReasonTimeout})
		return false, nil
	case <-ctx.Done():
		g.publishResolved(confirmID, ConfirmResponse{Approved: false, Reason: ConfirmReasonDenied})
		return false, ctx.Err()
	case <-g.done:
		g.publishResolved(confirmID, ConfirmResponse{Approved: false, Reason: ConfirmReasonDenied})
		return false, fmt.Errorf("confirmation cancelled")
	}
}

// beginWait prepares the gate for one confirmation request and returns its ID.
func (g *ConfirmGate) beginWait() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.resolved {
		g.id = uuid.NewString()
		g.resolved = false
		g.ch = make(chan ConfirmResponse, 1)
	}
	return g.id
}

// Resolve applies the client decision. Returns false when the gate is already resolved.
func (g *ConfirmGate) Resolve(approved bool, reason ConfirmReason) bool {
	g.mu.Lock()
	if g.resolved {
		g.mu.Unlock()
		return false
	}
	g.mu.Unlock()
	select {
	case g.ch <- ConfirmResponse{Approved: approved, Reason: reason}:
		return true
	default:
		return false
	}
}

// Cancel closes the gate without approving, used when the run is aborted.
func (g *ConfirmGate) Cancel() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.resolved {
		return
	}
	g.resolved = true
	close(g.done)
}

func (g *ConfirmGate) publishResolved(confirmID string, resp ConfirmResponse) {
	g.mu.Lock()
	if g.resolved {
		g.mu.Unlock()
		return
	}
	g.resolved = true
	g.mu.Unlock()

	if g.publisher == nil {
		return
	}
	reason := string(resp.Reason)
	if reason == "" {
		if resp.Approved {
			reason = string(ConfirmReasonApproved)
		} else {
			reason = string(ConfirmReasonDenied)
		}
	}
	_ = g.publisher.Publish(StreamEvent{
		Type:     EventTypeConfirmResolved,
		ID:       confirmID,
		Approved: resp.Approved,
		Reason:   reason,
	})
}

func formatConfirmSummary(event hooks.ToolCallEvent) string {
	switch event.ToolCall.Name {
	case "run_terminal":
		if cmd, ok := event.Args["command"]; ok {
			return fmt.Sprintf("command: %v", cmd)
		}
	case "write_file":
		if path, ok := event.Args["path"]; ok {
			return fmt.Sprintf("write file: %v", path)
		}
	case "run_code":
		if lang, ok := event.Args["language"]; ok {
			return fmt.Sprintf("run %v code", lang)
		}
	}
	return event.ToolCall.Name
}

// toolNeedsConfirm reports whether a tool requires explicit user approval in the Chat UI.
func toolNeedsConfirm(name string) bool {
	switch name {
	case "run_terminal", "write_file", "run_code":
		return true
	default:
		return false
	}
}
