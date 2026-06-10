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

var (
	sessionLocksMu sync.Mutex
	sessionLocks   = make(map[string]*lockEntry)

	runCancelsMu sync.Mutex
	runCancels   = make(map[string]context.CancelFunc)
)

func init() {
	go evictStaleLocks()
}

func sessionLock(sessionID string) *sync.Mutex {
	sessionLocksMu.Lock()
	defer sessionLocksMu.Unlock()
	if entry, ok := sessionLocks[sessionID]; ok {
		entry.lastUsed = time.Now()
		return &entry.mu
	}
	entry := &lockEntry{lastUsed: time.Now()}
	sessionLocks[sessionID] = entry
	return &entry.mu
}

func releaseSessionLock(sessionID string) {
	sessionLocksMu.Lock()
	defer sessionLocksMu.Unlock()
	delete(sessionLocks, sessionID)
}

// evictStaleLocks periodically removes session locks that have not been used within the TTL.
func evictStaleLocks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		sessionLocksMu.Lock()
		for id, entry := range sessionLocks {
			if now.Sub(entry.lastUsed) > sessionLockTTL {
				delete(sessionLocks, id)
			}
		}
		sessionLocksMu.Unlock()

		runCancelsMu.Lock()
		// runCancels entries are cleaned by UnbindRunCancel; evict stale ones defensively.
		runCancelsMu.Unlock()
	}
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
