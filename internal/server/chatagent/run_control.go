package chatagent

import (
	"context"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
)

const sessionLockTTL = 30 * time.Minute

type lockEntry struct {
	mu       sync.Mutex
	lastUsed time.Time
}

type runCancelEntry struct {
	cancel   context.CancelFunc
	lastUsed time.Time
}

func (s *Service) sessionLock(sessionID string) *sync.Mutex {
	s.sessionLocksMu.Lock()
	defer s.sessionLocksMu.Unlock()
	s.evictStaleSessionLocksLocked(time.Now())
	if entry, ok := s.sessionLocks[sessionID]; ok {
		entry.lastUsed = time.Now()
		return &entry.mu
	}
	entry := &lockEntry{lastUsed: time.Now()}
	s.sessionLocks[sessionID] = entry
	return &entry.mu
}

func (s *Service) releaseSessionLock(sessionID string) {
	s.sessionLocksMu.Lock()
	defer s.sessionLocksMu.Unlock()
	delete(s.sessionLocks, sessionID)
}

func (s *Service) evictStaleSessionLocksLocked(now time.Time) {
	for id, entry := range s.sessionLocks {
		if now.Sub(entry.lastUsed) > sessionLockTTL {
			delete(s.sessionLocks, id)
		}
	}
}

func (s *Service) evictStaleRunCancelsLocked(now time.Time) {
	for id, entry := range s.runCancels {
		if now.Sub(entry.lastUsed) > sessionLockTTL {
			delete(s.runCancels, id)
		}
	}
}

func (s *Service) registerRunCancel(sessionID string, cancel context.CancelFunc) {
	s.runCancelsMu.Lock()
	defer s.runCancelsMu.Unlock()
	s.evictStaleRunCancelsLocked(time.Now())
	if prev, ok := s.runCancels[sessionID]; ok {
		prev.cancel()
	}
	s.runCancels[sessionID] = &runCancelEntry{cancel: cancel, lastUsed: time.Now()}
}

func (s *Service) unregisterRunCancel(sessionID string) {
	s.runCancelsMu.Lock()
	defer s.runCancelsMu.Unlock()
	delete(s.runCancels, sessionID)
}

// BindRunCancel ties an agent run cancel function to a session for cooperative cancellation.
func (s *Service) BindRunCancel(sessionID string, cancel context.CancelFunc) {
	s.registerRunCancel(sessionID, cancel)
}

// UnbindRunCancel removes the run cancel function for a session.
func (s *Service) UnbindRunCancel(sessionID string) {
	s.unregisterRunCancel(sessionID)
}

func (s *Service) cancelRun(sessionID string) {
	s.runCancelsMu.Lock()
	s.evictStaleRunCancelsLocked(time.Now())
	entry, ok := s.runCancels[sessionID]
	if ok {
		entry.lastUsed = time.Now()
	}
	s.runCancelsMu.Unlock()
	if ok {
		flog.Info("[chat-agent] cancelled in-flight run session=%s", sessionID)
		entry.cancel()
	}
}
