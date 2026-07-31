package notify

import (
	"sync"
	"time"
)

var (
	presenceMu     sync.Mutex
	presenceByUID  = map[string]time.Time{}
	presenceWindow = 5 * time.Minute
	escalateAfter  = 10 * time.Minute
)

// SetPresenceWindowForTest overrides the presence activity window (tests only).
func SetPresenceWindowForTest(d time.Duration) {
	presenceMu.Lock()
	defer presenceMu.Unlock()
	presenceWindow = d
}

// SetEscalateAfterForTest overrides the default unread escalate delay (tests only).
func SetEscalateAfterForTest(d time.Duration) {
	presenceMu.Lock()
	defer presenceMu.Unlock()
	escalateAfter = d
}

// PresenceWindow returns the configured presence activity window.
func PresenceWindow() time.Duration {
	presenceMu.Lock()
	defer presenceMu.Unlock()
	return presenceWindow
}

// EscalateAfter returns the default unread escalate delay.
func EscalateAfter() time.Duration {
	presenceMu.Lock()
	defer presenceMu.Unlock()
	return escalateAfter
}

// TouchPresence records that uid was active on the web UI now.
func TouchPresence(uid string) {
	if uid == "" {
		return
	}
	presenceMu.Lock()
	defer presenceMu.Unlock()
	presenceByUID[uid] = time.Now()
}

// IsPresent reports whether uid has been active within the presence window.
func IsPresent(uid string) bool {
	if uid == "" {
		return false
	}
	presenceMu.Lock()
	defer presenceMu.Unlock()
	last, ok := presenceByUID[uid]
	if !ok {
		return false
	}
	return time.Since(last) <= presenceWindow
}

// ClearPresenceForTest removes all presence entries (tests only).
func ClearPresenceForTest() {
	presenceMu.Lock()
	defer presenceMu.Unlock()
	presenceByUID = map[string]time.Time{}
}
