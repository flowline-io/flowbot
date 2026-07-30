package store

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/config"
)

type stubDBOnlyAdapter struct {
	client *gen.Client
}

func (a *stubDBOnlyAdapter) Open(_ config.StoreType) error              { return nil }
func (a *stubDBOnlyAdapter) Close() error                              { return nil }
func (a *stubDBOnlyAdapter) IsOpen() bool                              { return true }
func (a *stubDBOnlyAdapter) GetName() string                           { return "stub-db-only" }
func (a *stubDBOnlyAdapter) Stats() any                                { return nil }
func (a *stubDBOnlyAdapter) Ping(_ context.Context) (time.Duration, error) {
	return 0, nil
}
func (a *stubDBOnlyAdapter) GetDB() any { return a.client }
func (a *stubDBOnlyAdapter) GetClient() *gen.Client {
	// Mimic BDD stubs that embed a nil Adapter: without an override this panics.
	// This stub intentionally returns nil so ClientFromDB must use GetDB.
	return nil
}

func TestClientFromDB_FallsBackToGetDB(t *testing.T) {
	orig := Database
	t.Cleanup(func() { Database = orig })

	sentinel := &gen.Client{}
	Database = &stubDBOnlyAdapter{client: sentinel}

	got := ClientFromDB()
	if got != sentinel {
		t.Fatalf("ClientFromDB() = %p, want GetDB client %p", got, sentinel)
	}
}

func TestClientFromDB_NilDatabase(t *testing.T) {
	orig := Database
	t.Cleanup(func() { Database = orig })
	Database = nil
	if ClientFromDB() != nil {
		t.Fatal("ClientFromDB() with nil Database should be nil")
	}
}
