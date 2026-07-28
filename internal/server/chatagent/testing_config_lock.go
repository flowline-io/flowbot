package chatagent

import (
	"sync"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
)

var (
	appConfigTestMu     sync.Mutex
	storeDatabaseTestMu sync.Mutex
)

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

// withIsolatedTestStore swaps store.Database for an in-memory SQLite adapter for the
// duration of the test. Access is serialized so parallel tests cannot race on the
// package-level store.Database pointer.
func withIsolatedTestStore(t *testing.T) {
	t.Helper()
	storeDatabaseTestMu.Lock()
	origDB := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() {
		store.Database = origDB
		storeDatabaseTestMu.Unlock()
	})
}
