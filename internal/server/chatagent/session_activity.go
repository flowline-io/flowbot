package chatagent

// SessionActivityRunning means an agent turn is in flight for the session.
const SessionActivityRunning = "running"

// SessionActivityNeedsApproval means the session is blocked on a tool confirmation.
const SessionActivityNeedsApproval = "needs_approval"

// HasPendingConfirm reports whether the session has an unresolved confirmation gate.
func (s *Service) HasPendingConfirm(sessionID string) bool {
	_, ok := s.LookupPendingConfirm(sessionID)
	return ok
}

// LookupPendingConfirm returns the outstanding confirm event for a session when waiting.
func (s *Service) LookupPendingConfirm(sessionID string) (StreamEvent, bool) {
	raw, ok := s.sessionConfirmGates.Load(sessionID)
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
func (s *Service) IsSessionRunning(sessionID string) bool {
	if _, ok := s.activeAPIRuns.Load(sessionID); ok {
		return true
	}
	s.runCancelsMu.Lock()
	defer s.runCancelsMu.Unlock()
	_, ok := s.runCancels[sessionID]
	return ok
}

// SessionActivity returns the list-facing runtime status for a session.
// Needs-approval takes precedence over running.
func (s *Service) SessionActivity(sessionID string) string {
	if s.HasPendingConfirm(sessionID) {
		return SessionActivityNeedsApproval
	}
	if s.IsSessionRunning(sessionID) {
		return SessionActivityRunning
	}
	return ""
}

// ListSessionIDsByActivity returns session IDs that currently match the activity filter.
func (s *Service) ListSessionIDsByActivity(activity string) []string {
	switch activity {
	case SessionActivityNeedsApproval:
		return s.listPendingConfirmSessionIDs()
	case SessionActivityRunning:
		return s.listRunningSessionIDs()
	default:
		return nil
	}
}

// CountPendingApprovalSessions returns how many sessions currently wait on tool approval.
func (s *Service) CountPendingApprovalSessions() int {
	return len(s.listPendingConfirmSessionIDs())
}

func (s *Service) listPendingConfirmSessionIDs() []string {
	out := make([]string, 0)
	s.sessionConfirmGates.Range(func(key, value any) bool {
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

func (s *Service) listRunningSessionIDs() []string {
	seen := make(map[string]struct{})
	s.activeAPIRuns.Range(func(key, _ any) bool {
		if sessionID, ok := key.(string); ok && sessionID != "" {
			seen[sessionID] = struct{}{}
		}
		return true
	})
	s.runCancelsMu.Lock()
	for sessionID := range s.runCancels {
		if sessionID != "" {
			seen[sessionID] = struct{}{}
		}
	}
	s.runCancelsMu.Unlock()
	out := make([]string, 0, len(seen))
	for sessionID := range seen {
		// Prefer needs_approval exclusivity for the running filter.
		if s.HasPendingConfirm(sessionID) {
			continue
		}
		out = append(out, sessionID)
	}
	return out
}
