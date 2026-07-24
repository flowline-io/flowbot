package chatagent

import "sync"

// harnessPoolMap exposes the harness pool map for package tests.
func (s *Service) harnessPoolMap() *sync.Map { return &s.harnessPool }

// activeAPIRunsMap exposes active API run state for package tests.
func (s *Service) activeAPIRunsMap() *sync.Map { return &s.activeAPIRuns }

// sessionConfirmGatesMap exposes confirm gates for package tests.
func (s *Service) sessionConfirmGatesMap() *sync.Map { return &s.sessionConfirmGates }

// runCancelsMap exposes run cancel entries for package tests.
func (s *Service) runCancelsMap() map[string]*runCancelEntry { return s.runCancels }

// runCancelsMutex exposes the run-cancel mutex for package tests.
func (s *Service) runCancelsMutex() *sync.Mutex { return &s.runCancelsMu }

// sessionLocksMap exposes session locks for package tests.
func (s *Service) sessionLocksMap() map[string]*lockEntry { return s.sessionLocks }

// sessionLocksMutex exposes the session-lock mutex for package tests.
func (s *Service) sessionLocksMutex() *sync.Mutex { return &s.sessionLocksMu }
