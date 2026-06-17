package chatagent

import (
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
)

// PermissionSessionManager keeps permission session state across runs for one chat session.
type PermissionSessionManager struct {
	states sync.Map
}

var permissionSessions PermissionSessionManager

// GetPermissionSession returns or creates session-scoped permission state.
func (m *PermissionSessionManager) GetPermissionSession(sessionID string) *permission.SessionState {
	if sessionID == "" {
		return permission.NewSessionState()
	}
	if raw, ok := m.states.Load(sessionID); ok {
		if state, ok := raw.(*permission.SessionState); ok {
			return state
		}
	}
	state := permission.NewSessionState()
	actual, _ := m.states.LoadOrStore(sessionID, state)
	if existing, ok := actual.(*permission.SessionState); ok {
		return existing
	}
	return state
}

// ClearPermissionSession removes session permission state.
func (m *PermissionSessionManager) ClearPermissionSession(sessionID string) {
	if sessionID == "" {
		return
	}
	m.states.Delete(sessionID)
}

// ResetPermissionSessionsForTest clears all in-memory permission session state.
func ResetPermissionSessionsForTest() {
	permissionSessions = PermissionSessionManager{}
}
