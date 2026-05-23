package store

import (
	"context"
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	_ "github.com/flowline-io/flowbot/internal/store/ent/gen/runtime"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

func getTestClient(t *testing.T) *gen.Client {
	t.Helper()
	client, err := gen.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed opening connection to sqlite: %v", err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed creating schema resources: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// ---------------------------------------------------------------------------
// AuditStore tests
// ---------------------------------------------------------------------------

func TestAuditStore_ImplementsAuditor(t *testing.T) {
	t.Parallel()
	var _ audit.Auditor = (*AuditStore)(nil)
}

func TestAuditStore_NilSafe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		store *AuditStore
	}{
		{name: "nil store", store: nil},
		{name: "zero-value store with nil client", store: &AuditStore{}},
		{name: "zero-value store", store: &AuditStore{client: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.Record(context.Background(), audit.Entry{
				Action: "test.action",
				Target: audit.Target{Type: "test", ID: "1"},
			})
			assert.NoError(t, err)
		})
	}
}

func TestAuditStore_RecordSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		wantErr bool
	}{
		{name: "nil store", store: nil, entry: audit.Entry{Action: "test.success"}, wantErr: false},
		{name: "zero-value store", store: &AuditStore{}, entry: audit.Entry{Action: "test.success"}, wantErr: false},
		{name: "nil client", store: &AuditStore{client: nil}, entry: audit.Entry{Action: "test.success"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordSuccess(context.Background(), tt.entry)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_RecordFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		err     error
		wantErr bool
	}{
		{name: "nil store with error", store: nil, entry: audit.Entry{Action: "test.fail"}, err: errors.New("boom"), wantErr: false},
		{name: "zero store with error", store: &AuditStore{}, entry: audit.Entry{Action: "test.fail"}, err: errors.New("boom"), wantErr: false},
		{name: "nil client with nil error", store: &AuditStore{client: nil}, entry: audit.Entry{Action: "test.fail"}, err: nil, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordFailure(context.Background(), tt.entry, tt.err)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_RecordRejected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		reason  string
		wantErr bool
	}{
		{name: "nil store", store: nil, entry: audit.Entry{Action: "test.deny"}, reason: "no permission", wantErr: false},
		{name: "zero store", store: &AuditStore{}, entry: audit.Entry{Action: "test.deny"}, reason: "no permission", wantErr: false},
		{name: "nil client", store: &AuditStore{client: nil}, entry: audit.Entry{Action: "test.deny"}, reason: "blocked", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordRejected(context.Background(), tt.entry, tt.reason)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_SubjectExtraction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		subject *audit.Subject
	}{
		{
			name: "full subject with user details",
			subject: &audit.Subject{
				SubjectType: "user",
				SubjectID:   "owner",
				UID:         "auth0|123",
				IPAddress:   "10.0.0.1",
				UserAgent:   "test/1.0",
			},
		},
		{
			name:    "nil subject",
			subject: nil,
		},
		{
			name: "system pipeline subject",
			subject: &audit.Subject{
				SubjectType: "pipeline",
				SubjectID:   "system:pipeline",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &AuditStore{}
			err := store.Record(context.Background(), audit.Entry{
				Subject: tt.subject,
				Action:  "test.action",
				Target:  audit.Target{Type: "test", ID: "1"},
			})
			assert.NoError(t, err)
		})
	}
}

// ---------------------------------------------------------------------------
// PollingStateStore tests
// ---------------------------------------------------------------------------

func TestPollingStateStore_LoadEmpty(t *testing.T) {
	t.Run("load empty database", func(t *testing.T) {
		client := getTestClient(t)
		store := NewPollingStateStore(client)

		state, err := store.LoadAll(context.Background())
		if err != nil {
			t.Fatalf("LoadAll: %v", err)
		}
		if len(state) != 0 {
			t.Fatalf("expected empty state, got %d entries", len(state))
		}
	})
}

func TestPollingStateStore_SaveAndLoad(t *testing.T) {
	client := getTestClient(t)
	store := NewPollingStateStore(client)

	tests := []struct {
		name     string
		resource string
		cursor   string
		hashes   map[string]string
	}{
		{
			name:     "single entry",
			resource: "github/starred",
			cursor:   "cursor-123",
			hashes:   map[string]string{"key1": "hash1", "key2": "hash2"},
		},
		{
			name:     "empty hashes",
			resource: "miniflux/entries",
			cursor:   "cursor-456",
			hashes:   map[string]string{},
		},
		{
			name:     "nil hashes saved as empty",
			resource: "gitea/issues",
			cursor:   "cursor-789",
			hashes:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Save(context.Background(), tt.resource, tt.cursor, tt.hashes)
			if err != nil {
				t.Fatalf("Save: %v", err)
			}

			loaded, err := store.LoadAll(context.Background())
			if err != nil {
				t.Fatalf("LoadAll: %v", err)
			}

			entry, ok := loaded[tt.resource]
			if !ok {
				t.Fatalf("expected entry for %s", tt.resource)
			}
			if entry.Cursor != tt.cursor {
				t.Errorf("cursor = %q, want %q", entry.Cursor, tt.cursor)
			}

			expectedLen := len(tt.hashes)
			if tt.hashes == nil {
				expectedLen = 0
			}
			if len(entry.KnownHashes) != expectedLen {
				t.Errorf("known_hashes len = %d, want %d", len(entry.KnownHashes), expectedLen)
			}
			for k, v := range tt.hashes {
				if got := entry.KnownHashes[k]; got != v {
					t.Errorf("known_hashes[%q] = %q, want %q", k, got, v)
				}
			}
		})
	}
}

func TestPollingStateStore_Update(t *testing.T) {
	client := getTestClient(t)
	store := NewPollingStateStore(client)

	tests := []struct {
		name      string
		first     map[string]string
		second    map[string]string
		wantFinal map[string]string
	}{
		{
			name:      "replace all entries",
			first:     map[string]string{"a": "h1", "b": "h2"},
			second:    map[string]string{"c": "h3", "d": "h4"},
			wantFinal: map[string]string{"c": "h3", "d": "h4"},
		},
		{
			name:      "update existing entry",
			first:     map[string]string{"a": "h1", "b": "h2"},
			second:    map[string]string{"b": "h2-new"},
			wantFinal: map[string]string{"b": "h2-new"},
		},
		{
			name:      "clear all entries",
			first:     map[string]string{"a": "h1"},
			second:    map[string]string{},
			wantFinal: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := store.Save(ctx, "test/rsrc", "cursor-1", tt.first); err != nil {
				t.Fatalf("first Save: %v", err)
			}
			if err := store.Save(ctx, "test/rsrc", "cursor-2", tt.second); err != nil {
				t.Fatalf("second Save: %v", err)
			}

			loaded, err := store.LoadAll(ctx)
			if err != nil {
				t.Fatalf("LoadAll: %v", err)
			}
			entry := loaded["test/rsrc"]
			if len(entry.KnownHashes) != len(tt.wantFinal) {
				t.Errorf("known_hashes len = %d, want %d", len(entry.KnownHashes), len(tt.wantFinal))
			}
			for k, v := range tt.wantFinal {
				if got := entry.KnownHashes[k]; got != v {
					t.Errorf("known_hashes[%q] = %q, want %q", k, got, v)
				}
			}
		})
	}
}
