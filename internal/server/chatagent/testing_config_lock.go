package chatagent

import (
	"sync"
	"testing"
)

var appConfigTestMu sync.Mutex

// LockAppConfigForTest serializes reads and writes of config.App during parallel tests.
func LockAppConfigForTest(t *testing.T) {
	t.Helper()
	appConfigTestMu.Lock()
	t.Cleanup(func() {
		appConfigTestMu.Unlock()
	})
}
