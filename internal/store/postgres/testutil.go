package postgres

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/sqlitetest"
)

// NewSQLiteTestAdapter returns an in-memory SQLite-backed store adapter for tests.
func NewSQLiteTestAdapter(t *testing.T) store.Adapter {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	return &adapter{client: sqlitetest.OpenClient(t, dbName)}
}
