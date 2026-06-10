package chatagent

import (
	"context"
	"sync"

	"github.com/flowline-io/flowbot/pkg/flog"
)

var (
	sessionLocksMu sync.Mutex
	sessionLocks   = make(map[string]*sync.Mutex)

	runCancelsMu sync.Mutex
	runCancels   = make(map[string]context.CancelFunc)
)

func sessionLock(sessionID string) *sync.Mutex {
	sessionLocksMu.Lock()
	defer sessionLocksMu.Unlock()
	if lock, ok := sessionLocks[sessionID]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	sessionLocks[sessionID] = lock
	return lock
}

func releaseSessionLock(sessionID string) {
	sessionLocksMu.Lock()
	defer sessionLocksMu.Unlock()
	delete(sessionLocks, sessionID)
}

func registerRunCancel(sessionID string, cancel context.CancelFunc) {
	runCancelsMu.Lock()
	defer runCancelsMu.Unlock()
	if prev, ok := runCancels[sessionID]; ok {
		prev()
	}
	runCancels[sessionID] = cancel
}

func unregisterRunCancel(sessionID string) {
	runCancelsMu.Lock()
	defer runCancelsMu.Unlock()
	delete(runCancels, sessionID)
}

// BindRunCancel ties an agent run cancel function to a session for cooperative cancellation.
func BindRunCancel(sessionID string, cancel context.CancelFunc) {
	registerRunCancel(sessionID, cancel)
}

// UnbindRunCancel removes the run cancel function for a session.
func UnbindRunCancel(sessionID string) {
	unregisterRunCancel(sessionID)
}

func cancelRun(sessionID string) {
	runCancelsMu.Lock()
	cancel, ok := runCancels[sessionID]
	runCancelsMu.Unlock()
	if ok {
		flog.Info("[chat-agent] cancelled in-flight run session=%s", sessionID)
		cancel()
	}
}
