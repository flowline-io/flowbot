// Package sqlitetest opens in-memory SQLite databases for unit tests using modernc.org/sqlite.
package sqlitetest

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"modernc.org/sqlite"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	// required by schema hooks.
	_ "github.com/flowline-io/flowbot/internal/store/ent/gen/runtime"
)

var (
	schemaTemplateOnce sync.Once
	schemaTemplatePath string
	schemaTemplateErr  error
	openSeq            atomic.Uint64
)

// OpenClient opens a private in-memory SQLite database and returns an ent client with schema applied.
// dbName is a caller hint only; a unique suffix is always appended so shared-memory SQLite
// databases never collide across parallel tests or -count>1 (raw CREATE TABLE is not idempotent).
//
// Schema is materialized once into a temp-file template, then each OpenClient clones it via
// SQLite's Online Backup API (NewRestore). Replaying hundreds of DDL statements under -race
// (or calling Atlas Schema.Create per test) is prohibitively slow for packages such as
// internal/modules/web.
func OpenClient(t *testing.T, dbName string) *gen.Client {
	t.Helper()

	templatePath, err := cachedSchemaTemplate()
	if err != nil {
		t.Fatalf("failed preparing schema template: %v", err)
	}

	unique := fmt.Sprintf("%s_%d", sanitizeDBName(dbName), openSeq.Add(1))
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", unique)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed opening connection to sqlite: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := restoreSchemaTemplate(sqlDB, templatePath); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("failed restoring schema template: %v", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("failed enabling foreign keys: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, sqlDB)
	client := gen.NewClient(gen.Driver(drv))
	t.Cleanup(func() { client.Close() })
	return client
}

func sanitizeDBName(dbName string) string {
	if dbName == "" {
		return "ent"
	}
	return strings.NewReplacer("/", "_", " ", "_", "?", "_", "&", "_", "=", "_").Replace(dbName)
}

func cachedSchemaTemplate() (string, error) {
	schemaTemplateOnce.Do(func() {
		dir, err := os.MkdirTemp("", "sqlitetest-schema-*")
		if err != nil {
			schemaTemplateErr = err
			return
		}
		path := filepath.Join(dir, "schema.db")

		sqlDB, err := sql.Open("sqlite", path)
		if err != nil {
			schemaTemplateErr = err
			return
		}
		sqlDB.SetMaxOpenConns(1)
		if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
			_ = sqlDB.Close()
			schemaTemplateErr = err
			return
		}

		drv := entsql.OpenDB(dialect.SQLite, sqlDB)
		client := gen.NewClient(gen.Driver(drv))

		var buf bytes.Buffer
		if err := client.Schema.WriteTo(context.Background(), &buf); err != nil {
			_ = client.Close()
			_ = sqlDB.Close()
			schemaTemplateErr = err
			return
		}
		stmts := splitSQLStatements(buf.String())
		if len(stmts) == 0 {
			_ = client.Close()
			_ = sqlDB.Close()
			schemaTemplateErr = fmt.Errorf("ent schema WriteTo produced no statements")
			return
		}
		for _, stmt := range stmts {
			if _, err := sqlDB.Exec(stmt); err != nil {
				_ = client.Close()
				_ = sqlDB.Close()
				schemaTemplateErr = fmt.Errorf("apply schema statement %q: %w", truncateSQL(stmt), err)
				return
			}
		}
		if err := client.Close(); err != nil {
			_ = sqlDB.Close()
			schemaTemplateErr = err
			return
		}
		if err := sqlDB.Close(); err != nil {
			schemaTemplateErr = err
			return
		}
		schemaTemplatePath = path
	})
	return schemaTemplatePath, schemaTemplateErr
}

func restoreSchemaTemplate(sqlDB *sql.DB, templatePath string) error {
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Raw(func(driverConn any) error {
		type restorer interface {
			NewRestore(string) (*sqlite.Backup, error)
		}
		r, ok := driverConn.(restorer)
		if !ok {
			return fmt.Errorf("sqlite driver %T does not support NewRestore", driverConn)
		}
		bk, err := r.NewRestore(templatePath)
		if err != nil {
			return err
		}
		for {
			more, err := bk.Step(-1)
			if err != nil {
				_ = bk.Finish()
				return err
			}
			if !more {
				break
			}
		}
		return bk.Finish()
	})
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
