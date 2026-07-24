package chatagent

import "sync"

// BindSharedService installs the production chatagent Service used by scheduled
// and pipeline run entry points. Call once at server bootstrap before traffic.
// Passing nil clears the binding (tests only).
func BindSharedService(s *Service) {
	scheduledRunService = s
	pipelineRunService = s
}

// ResetHotPathForTest clears all Service-owned hot-path runtime state.
func (s *Service) ResetHotPathForTest() {
	if s == nil {
		return
	}
	s.ResetHarnessPoolForTest()
	s.ResetSessionEventHubsForTest()
	s.ResetPermissionSessionsForTest()
	s.sessionLocksMu.Lock()
	s.sessionLocks = make(map[string]*lockEntry)
	s.sessionLocksMu.Unlock()
	s.runCancelsMu.Lock()
	s.runCancels = make(map[string]*runCancelEntry)
	s.runCancelsMu.Unlock()
	s.sessionConfirmGates = sync.Map{}
	s.activeAPIRuns = sync.Map{}
}
