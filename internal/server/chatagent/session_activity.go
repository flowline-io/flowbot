package chatagent

// SessionActivityRunning means an agent turn is in flight for the session.
const SessionActivityRunning = "running"

// SessionActivityNeedsApproval means the session is blocked on a tool confirmation.
const SessionActivityNeedsApproval = "needs_approval"

// HasPendingConfirm reports whether the session has an unresolved confirmation gate.
func HasPendingConfirm(sessionID string) bool {
	_, ok := LookupPendingConfirm(sessionID)
	return ok
}

// LookupPendingConfirm returns the outstanding confirm event for a session when waiting.
func LookupPendingConfirm(sessionID string) (StreamEvent, bool) {
	raw, ok := sessionConfirmGates.Load(sessionID)
	if !ok {
		return StreamEvent{}, false
	}
	gate, ok := raw.(*ConfirmGate)
	if !ok {
		return StreamEvent{}, false
	}
	return gate.PendingEvent()
}

// IsSessionRunning reports whether the session has an in-flight agent run.
func IsSessionRunning(sessionID string) bool {
	if _, ok := activeAPIRuns.Load(sessionID); ok {
		return true
	}
	runCancelsMu.Lock()
	defer runCancelsMu.Unlock()
	_, ok := runCancels[sessionID]
	return ok
}

// SessionActivity returns the list-facing runtime status for a session.
// Needs-approval takes precedence over running.
func SessionActivity(sessionID string) string {
	if HasPendingConfirm(sessionID) {
		return SessionActivityNeedsApproval
	}
	if IsSessionRunning(sessionID) {
		return SessionActivityRunning
	}
	return ""
}

// ListSessionIDsByActivity returns session IDs that currently match the activity filter.
func ListSessionIDsByActivity(activity string) []string {
	switch activity {
	case SessionActivityNeedsApproval:
		return listPendingConfirmSessionIDs()
	case SessionActivityRunning:
		return listRunningSessionIDs()
	default:
		return nil
	}
}

func listPendingConfirmSessionIDs() []string {
	out := make([]string, 0)
	sessionConfirmGates.Range(func(key, value any) bool {
		sessionID, ok := key.(string)
		if !ok || sessionID == "" {
			return true
		}
		gate, ok := value.(*ConfirmGate)
		if ok && gate.IsWaiting() {
			out = append(out, sessionID)
		}
		return true
	})
	return out
}

func listRunningSessionIDs() []string {
	seen := make(map[string]struct{})
	activeAPIRuns.Range(func(key, _ any) bool {
		if sessionID, ok := key.(string); ok && sessionID != "" {
			seen[sessionID] = struct{}{}
		}
		return true
	})
	runCancelsMu.Lock()
	for sessionID := range runCancels {
		if sessionID != "" {
			seen[sessionID] = struct{}{}
		}
	}
	runCancelsMu.Unlock()
	out := make([]string, 0, len(seen))
	for sessionID := range seen {
		// Prefer needs_approval exclusivity for the running filter.
		if HasPendingConfirm(sessionID) {
			continue
		}
		out = append(out, sessionID)
	}
	return out
}
