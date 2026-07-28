// Package sqlitetest opens in-memory SQLite databases for unit tests using modernc.org/sqlite.
package sqlitetest

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
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

var (
	schemaDDLOnce sync.Once
	schemaDDL     []string
	schemaDDLErr  error
)

// OpenClient opens a private in-memory SQLite database and returns an ent client with schema applied.
// dbName isolates databases when tests run in parallel (use t.Name() or a stable per-suite name).
//
// Schema DDL is generated once via ent WriteTo and replayed with Exec. Repeated Atlas
// Schema.Create under -race is prohibitively slow when many tests open SQLite adapters.
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

	stmts, err := cachedSchemaDDL()
	if err != nil {
		t.Fatalf("failed preparing schema DDL: %v", err)
	}
	for _, stmt := range stmts {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("failed applying schema statement %q: %v", truncateSQL(stmt), err)
		}
	}

	drv := entsql.OpenDB(dialect.SQLite, sqlDB)
	client := gen.NewClient(gen.Driver(drv))
	t.Cleanup(func() { client.Close() })
	return client
}

func cachedSchemaDDL() ([]string, error) {
	schemaDDLOnce.Do(func() {
		sqlDB, err := sql.Open("sqlite", "file:sqlitetest_schema_ddl?mode=memory&cache=shared")
		if err != nil {
			schemaDDLErr = err
			return
		}
		defer sqlDB.Close()
		sqlDB.SetMaxOpenConns(1)
		if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
			schemaDDLErr = err
			return
		}

		drv := entsql.OpenDB(dialect.SQLite, sqlDB)
		client := gen.NewClient(gen.Driver(drv))
		defer client.Close()

		var buf bytes.Buffer
		if err := client.Schema.WriteTo(context.Background(), &buf); err != nil {
			schemaDDLErr = err
			return
		}
		schemaDDL = splitSQLStatements(buf.String())
		if len(schemaDDL) == 0 {
			schemaDDLErr = fmt.Errorf("ent schema WriteTo produced no statements")
		}
	})
	return schemaDDL, schemaDDLErr
}

func splitSQLStatements(script string) []string {
	parts := strings.Split(script, ";\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		stmt = strings.TrimSuffix(stmt, ";")
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		out = append(out, stmt)
	}
	return out
}

func truncateSQL(stmt string) string {
	const maxLen = 80
	stmt = strings.ReplaceAll(stmt, "\n", " ")
	if len(stmt) <= maxLen {
		return stmt
	}
	return stmt[:maxLen] + "..."
}
