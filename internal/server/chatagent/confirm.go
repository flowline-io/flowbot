package chatagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
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

// ConfirmMode is how the user resolved an approval prompt.
type ConfirmMode string

const (
	ConfirmModeOnce   ConfirmMode = "once"
	ConfirmModeAlways ConfirmMode = "always"
	ConfirmModeReject ConfirmMode = "reject"
)

// ConfirmResponse is the user's answer to a pending tool confirmation.
type ConfirmResponse struct {
	Approved bool
	Reason   ConfirmReason
	Mode     ConfirmMode
	Pattern  string
}

// ConfirmGate blocks tool execution until the client approves or denies.
type ConfirmGate struct {
	mu        sync.Mutex
	id        string
	sessionID string
	publisher EventPublisher
	ch        chan ConfirmResponse
	done      chan struct{}
	resolved  bool
	waiting   bool
	timeout   time.Duration
	pending   *StreamEvent
}

// NewConfirmGate creates a gate that publishes confirm events to session subscribers.
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

// IsWaiting reports whether the gate is currently blocking on a client decision.
func (g *ConfirmGate) IsWaiting() bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.waiting
}

// PendingEvent returns the last published confirm payload while the gate is waiting.
func (g *ConfirmGate) PendingEvent() (StreamEvent, bool) {
	if g == nil {
		return StreamEvent{}, false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.waiting || g.pending == nil {
		return StreamEvent{}, false
	}
	return *g.pending, true
}

// Wait publishes a confirm event and blocks until the client responds or times out.
func (g *ConfirmGate) Wait(ctx context.Context, event hooks.ToolCallEvent, eval permission.Result) (ConfirmResponse, error) {
	confirmID := g.beginWait()
	summary := formatConfirmSummary(event)
	payload := StreamEvent{
		Type:             EventTypeConfirm,
		ID:               confirmID,
		Tool:             event.ToolCall.Name,
		Summary:          summary,
		Permission:       eval.PermissionKey,
		Pattern:          eval.Pattern,
		SuggestedPattern: eval.SuggestedPattern,
		SuggestAlways:    eval.SuggestAlways,
	}
	g.setPending(payload)
	_ = g.emit(payload)

	timer := time.NewTimer(g.timeout)
	defer timer.Stop()

	select {
	case resp := <-g.ch:
		g.publishResolved(confirmID, resp)
		return resp, nil
	case <-timer.C:
		resp := ConfirmResponse{Approved: false, Reason: ConfirmReasonTimeout, Mode: ConfirmModeReject}
		g.publishResolved(confirmID, resp)
		return resp, nil
	case <-ctx.Done():
		resp := ConfirmResponse{Approved: false, Reason: ConfirmReasonDenied, Mode: ConfirmModeReject}
		g.publishResolved(confirmID, resp)
		return resp, ctx.Err()
	case <-g.done:
		resp := ConfirmResponse{Approved: false, Reason: ConfirmReasonDenied, Mode: ConfirmModeReject}
		g.publishResolved(confirmID, resp)
		return resp, fmt.Errorf("confirmation cancelled")
	}
}

func (g *ConfirmGate) setPending(event StreamEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	cp := event
	g.pending = &cp
}

func (g *ConfirmGate) clearPending() {
	g.pending = nil
}

func (g *ConfirmGate) beginWait() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.resolved {
		g.id = uuid.NewString()
		g.resolved = false
		g.ch = make(chan ConfirmResponse, 1)
		g.done = make(chan struct{})
	}
	g.waiting = true
	return g.id
}

// Resolve applies the client decision. Returns false when the gate is already resolved.
func (g *ConfirmGate) Resolve(resp ConfirmResponse) bool {
	g.mu.Lock()
	if g.resolved {
		g.mu.Unlock()
		return false
	}
	g.mu.Unlock()
	select {
	case g.ch <- resp:
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
	g.waiting = false
	g.clearPending()
	close(g.done)
}

func (g *ConfirmGate) publishResolved(confirmID string, resp ConfirmResponse) {
	g.mu.Lock()
	if g.resolved {
		g.mu.Unlock()
		return
	}
	g.resolved = true
	g.waiting = false
	g.clearPending()
	g.mu.Unlock()

	reason := string(resp.Reason)
	if reason == "" {
		if resp.Approved {
			reason = string(ConfirmReasonApproved)
		} else {
			reason = string(ConfirmReasonDenied)
		}
	}
	event := StreamEvent{
		Type:     EventTypeConfirmResolved,
		ID:       confirmID,
		Approved: resp.Approved,
		Reason:   reason,
		Mode:     string(resp.Mode),
	}
	_ = g.emit(event)
}

func (g *ConfirmGate) emit(event StreamEvent) error {
	if g.publisher != nil {
		return g.publisher.Publish(event)
	}
	GetSessionEventHub(g.sessionID).publish(event)
	return nil
}

// alwaysGrantPattern resolves the pattern stored for ConfirmModeAlways.
// It rejects client patterns that differ from the server suggestion.
func alwaysGrantPattern(eval permission.Result, clientPattern string) (string, bool) {
	if !eval.SuggestAlways || eval.SuggestedPattern == "" {
		return "", false
	}
	pattern := strings.TrimSpace(clientPattern)
	if pattern == "" {
		return eval.SuggestedPattern, true
	}
	if pattern != eval.SuggestedPattern {
		return "", false
	}
	return eval.SuggestedPattern, true
}

func formatConfirmSummary(event hooks.ToolCallEvent) string {
	switch event.ToolCall.Name {
	case permission.ToolRunTerminal:
		if cmd, ok := event.Args["command"]; ok {
			return fmt.Sprintf("command: %v", cmd)
		}
	case permission.ToolWriteFile:
		if path, ok := event.Args["path"]; ok {
			return fmt.Sprintf("write file: %v", path)
		}
	case permission.ToolRunCode:
		if lang, ok := event.Args["language"]; ok {
			return fmt.Sprintf("run %v code", lang)
		}
	case permission.ToolReadFile:
		if path, ok := event.Args["path"]; ok {
			return fmt.Sprintf("read file: %v", path)
		}
	case permission.ToolWebSearch:
		if query, ok := event.Args["query"]; ok {
			return fmt.Sprintf("search: %v", query)
		}
	case permission.ToolReadSkill:
		if name, ok := event.Args["name"]; ok {
			return fmt.Sprintf("skill: %v", name)
		}
	}
	return event.ToolCall.Name
}
