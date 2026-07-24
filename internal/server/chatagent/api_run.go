package chatagent

// APIRunOptions configures an HTTP Chat Agent run with SSE publishing and confirmations.
type APIRunOptions struct {
	Publisher EventPublisher
	Confirm   *ConfirmGate
	OwnerUID  string
}

// APIRunState tracks one in-flight HTTP run for cancel and confirm routing.
type APIRunState struct {
	gate      *ConfirmGate
	publisher *ChannelPublisher
}

// NewAPIRunState builds run state for an active SSE connection.
func NewAPIRunState(publisher *ChannelPublisher, gate *ConfirmGate) *APIRunState {
	return &APIRunState{gate: gate, publisher: publisher}
}

// Publisher returns the SSE publisher when present.
func (s *APIRunState) Publisher() EventPublisher {
	if s == nil {
		return nil
	}
	return s.publisher
}

// TrySetAPIRunState registers run state when no other run is active for the session.
func (s *Service) TrySetAPIRunState(sessionID string, state *APIRunState) error {
	if state == nil {
		return ErrRunInFlight
	}
	if _, loaded := s.activeAPIRuns.LoadOrStore(sessionID, state); loaded {
		return ErrRunInFlight
	}
	if state.gate != nil {
		s.sessionConfirmGates.Store(sessionID, state.gate)
	}
	return nil
}

// ClearAPIRunState removes run state only when it matches the active connection.
func (s *Service) ClearAPIRunState(sessionID string, expected *APIRunState) {
	if expected != nil {
		s.activeAPIRuns.CompareAndDelete(sessionID, expected)
		if expected.gate != nil {
			s.sessionConfirmGates.CompareAndDelete(sessionID, expected.gate)
		}
		return
	}
	if raw, ok := s.activeAPIRuns.LoadAndDelete(sessionID); ok {
		if state, ok := raw.(*APIRunState); ok && state.gate != nil {
			state.gate.Cancel()
			s.sessionConfirmGates.CompareAndDelete(sessionID, state.gate)
		}
	}
}

// GetAPIRunState returns the active HTTP run state when present.
func (s *Service) GetAPIRunState(sessionID string) (*APIRunState, bool) {
	raw, ok := s.activeAPIRuns.Load(sessionID)
	if !ok {
		return nil, false
	}
	state, ok := raw.(*APIRunState)
	return state, ok
}

// CancelSessionRun aborts the in-flight agent run for a session.
func (s *Service) CancelSessionRun(sessionID string) {
	s.AbortSessionHarness(sessionID)
	s.cancelRun(sessionID)
}

// ResolveConfirm applies a client confirmation for the active gate on a session.
func (s *Service) ResolveConfirm(sessionID, confirmID string, approved bool, mode ConfirmMode, pattern string, reason ConfirmReason) (bool, error) {
	raw, ok := s.sessionConfirmGates.Load(sessionID)
	if !ok {
		return false, ErrConfirmNotFound
	}
	gate, ok := raw.(*ConfirmGate)
	if !ok || gate.ID() != confirmID {
		return false, ErrConfirmNotFound
	}
	if mode == "" {
		if approved {
			mode = ConfirmModeOnce
		} else {
			mode = ConfirmModeReject
		}
	}
	resp := ConfirmResponse{Approved: approved, Reason: reason, Mode: mode, Pattern: pattern}
	if !gate.Resolve(resp) {
		return false, ErrConfirmResolved
	}
	return true, nil
}
