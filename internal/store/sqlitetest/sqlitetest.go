// Package sqlitetest opens in-memory SQLite databases for unit tests using modernc.org/sqlite.
package sqlitetest

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	// required by schema hooks.
	_ "github.com/flowline-io/flowbot/internal/store/ent/gen/runtime"

	// register modernc sqlite driver for in-memory test databases.
	_ "modernc.org/sqlite"
)

// schemaMu serializes ent schema creation to avoid data races in ent's internal migration code.
var schemaMu sync.Mutex

// OpenClient opens a private in-memory SQLite database and returns an ent client with schema applied.
// dbName isolates databases when tests run in parallel (use t.Name() or a stable per-suite name).
func OpenClient(t *testing.T, dbName string) *gen.Client {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	if err != nil {
		t.Fatalf("failed opening connection to sqlite: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("failed enabling foreign keys: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, sqlDB)
	client := gen.NewClient(gen.Driver(drv))

	schemaMu.Lock()
	err = client.Schema.Create(context.Background())
	schemaMu.Unlock()
	if err != nil {
		t.Fatalf("failed creating schema resources: %v", err)
	}

	t.Cleanup(func() { client.Close() })
	return client
}
