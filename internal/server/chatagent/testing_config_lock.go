package chatagent

import (
	"sync"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/dcg"
)

var appConfigTestMu sync.Mutex

// LockAppConfigForTest serializes reads and writes of config.App during parallel tests.
// It also installs an AllowAll DCG checker so existing hook tests are not fail-closed
// when the process default is uninitialized or ErrorChecker.
func LockAppConfigForTest(t *testing.T) {
	t.Helper()
	appConfigTestMu.Lock()
	prev := dcg.DefaultChecker()
	dcg.SetDefaultChecker(dcg.AllowAllChecker{})
	t.Cleanup(func() {
		dcg.SetDefaultChecker(prev)
		appConfigTestMu.Unlock()
	})
}
