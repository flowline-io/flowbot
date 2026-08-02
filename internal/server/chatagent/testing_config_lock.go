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

// installTestDatabase replaces store.Database for the test lifetime.
// It holds storeDatabaseTestMu for the whole test and drains approval-notify
// goroutines before each swap so parallel tests cannot race the package-level
// adapter pointer.
func installTestDatabase(t *testing.T, db store.Adapter) {
	t.Helper()
	storeDatabaseTestMu.Lock()
	WaitApprovalNotifyForTest()
	orig := store.Database
	store.Database = db
	if db != nil {
		wireNotifyStoresForTest(t)
	} else {
		clearNotifyStoresForTest(t)
	}
	t.Cleanup(func() {
		WaitApprovalNotifyForTest()
		store.Database = orig
		storeDatabaseTestMu.Unlock()
	})
}

// installSQLiteTestDatabase installs an in-memory SQLite adapter for the test.
func installSQLiteTestDatabase(t *testing.T) {
	t.Helper()
	installTestDatabase(t, postgres.NewSQLiteTestAdapter(t))
}

// InstallSQLiteTestDatabaseForTest installs an in-memory SQLite adapter for tests
// in external packages (package chatagent_test).
func InstallSQLiteTestDatabaseForTest(t *testing.T) {
	installSQLiteTestDatabase(t)
}

// InstallTestDatabaseForTest replaces store.Database for the test lifetime.
func InstallTestDatabaseForTest(t *testing.T, db store.Adapter) {
	installTestDatabase(t, db)
}

// restoreTestDatabase drains approval notify then assigns store.Database.
// Callers must already hold storeDatabaseTestMu via installTestDatabase.
func restoreTestDatabase(db store.Adapter) {
	WaitApprovalNotifyForTest()
	store.Database = db
}

// lockStoreDatabaseForTest serializes store.Database access for tests that read it
// (for example ConfirmGate approval notify) without installing a replacement.
func lockStoreDatabaseForTest(t *testing.T) {
	t.Helper()
	storeDatabaseTestMu.Lock()
	t.Cleanup(func() {
		WaitApprovalNotifyForTest()
		storeDatabaseTestMu.Unlock()
	})
}

// withIsolatedTestStore swaps store.Database for an in-memory SQLite adapter for the
// duration of the test. Access is serialized so parallel tests cannot race on the
// package-level store.Database pointer.
func withIsolatedTestStore(t *testing.T) {
	t.Helper()
	installSQLiteTestDatabase(t)
}
