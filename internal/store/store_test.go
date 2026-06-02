package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	_ "github.com/flowline-io/flowbot/internal/store/ent/gen/runtime"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestSanitizeAuditValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  any
		expect any
	}{
		{
			name:   "nil input",
			input:  nil,
			expect: nil,
		},
		{
			name:   "string passes through",
			input:  "hello",
			expect: "hello",
		},
		{
			name:   "int passes through",
			input:  42,
			expect: 42,
		},
		{
			name: "redacts password and token keys",
			input: map[string]any{
				"username": "alice",
				"password": "secret123",
				"token":    "abc.def.ghi",
				"api_key":  "key-1234",
				"action":   "login",
			},
			expect: map[string]any{
				"username": "alice",
				"password": "[redacted]",
				"token":    "[redacted]",
				"api_key":  "[redacted]",
				"action":   "login",
			},
		},
		{
			name: "case-insensitive redaction",
			input: map[string]any{
				"PASSWORD": "MyP@ss",
				"Password": "AnotherP@ss",
			},
			expect: map[string]any{
				"PASSWORD": "[redacted]",
				"Password": "[redacted]",
			},
		},
		{
			name: "recursively redacts nested maps",
			input: map[string]any{
				"user": map[string]any{
					"name":     "bob",
					"password": "nested-secret",
				},
			},
			expect: map[string]any{
				"user": map[string]any{
					"name":     "bob",
					"password": "[redacted]",
				},
			},
		},
		{
			name: "redacts in arrays",
			input: []any{
				map[string]any{"id": "1", "secret": "s1"},
				map[string]any{"id": "2", "secret": "s2"},
			},
			expect: []any{
				map[string]any{"id": "1", "secret": "[redacted]"},
				map[string]any{"id": "2", "secret": "[redacted]"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeAuditValue(tt.input)
			assert.Equal(t, tt.expect, got)
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

// ---------------------------------------------------------------------------
// PipelineStore tests
// ---------------------------------------------------------------------------

func TestPipelineDefinitionStore_CreateAndGet(t *testing.T) {
	client := getTestClient(t)
	store := NewPipelineStore(client)

	tests := []struct {
		name         string
		pipelineName string
		description  string
		wantErr      bool
	}{
		{
			name:         "happy path - create pipeline",
			pipelineName: "test-pipeline",
			description:  "A test pipeline",
			wantErr:      false,
		},
		{
			name:         "empty description is ok",
			pipelineName: "no-desc-pipeline",
			description:  "",
			wantErr:      false,
		},
		{
			name:         "duplicate name returns error",
			pipelineName: "test-pipeline",
			description:  "duplicate",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := store.CreateDefinition(ctx, tt.pipelineName, tt.description)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			def, err := store.GetDefinitionByName(ctx, tt.pipelineName)
			require.NoError(t, err)
			assert.Equal(t, tt.pipelineName, def.Name)
			assert.Equal(t, tt.description, def.Description)
			assert.Empty(t, def.YamlDraft)
			assert.Equal(t, 1, def.Version)
		})
	}
}

func TestPipelineDefinitionStore_UpdateDraftConcurrency(t *testing.T) {
	client := getTestClient(t)
	store := NewPipelineStore(client)

	ctx := context.Background()
	err := store.CreateDefinition(ctx, "concurrent-test", "")
	require.NoError(t, err)

	// Update with version 1 — should succeed, version becomes 2
	def, err := store.UpdateDefinitionDraft(ctx, "concurrent-test", "steps: []", 1)
	require.NoError(t, err)
	assert.Equal(t, 2, def.Version)

	// Update with stale version 1 — should fail with ErrConflict
	_, err = store.UpdateDefinitionDraft(ctx, "concurrent-test", "steps: [a]", 1)
	require.ErrorIs(t, err, types.ErrConflict)

	// Update with current version 2 — should succeed
	def, err = store.UpdateDefinitionDraft(ctx, "concurrent-test", "steps: [b]", 2)
	require.NoError(t, err)
	assert.Equal(t, 3, def.Version)
}

func TestPipelineDefinitionStore_PublishAndListPublished(t *testing.T) {
	client := getTestClient(t)
	store := NewPipelineStore(client)

	ctx := context.Background()
	require.NoError(t, store.CreateDefinition(ctx, "pub-test", "desc"))

	// Set draft then publish with version 1
	_, err := store.UpdateDefinitionDraft(ctx, "pub-test", "name: pub-test\ntriggers: []\nsteps: []", 1)
	require.NoError(t, err)

	// Publish with version 2 (the new version after update)
	def, err := store.PublishDefinition(ctx, "pub-test", 2)
	require.NoError(t, err)
	assert.Equal(t, "published", string(def.Status))

	// Publish with stale version should fail
	_, err = store.PublishDefinition(ctx, "pub-test", 1)
	require.Error(t, err)

	// List published should return 1 record
	defs, err := store.ListPublishedDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, defs, 1)
	assert.Equal(t, "pub-test", defs[0].Name)
}

func TestPipelineDefinitionStore_ListAndDelete(t *testing.T) {
	client := getTestClient(t)
	store := NewPipelineStore(client)

	ctx := context.Background()
	require.NoError(t, store.CreateDefinition(ctx, "list-1", ""))
	require.NoError(t, store.CreateDefinition(ctx, "list-2", ""))
	require.NoError(t, store.CreateDefinition(ctx, "list-3", ""))

	// List returns all 3
	defs, err := store.ListDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, defs, 3)

	// Delete list-1
	count, err := store.DeleteDefinitionByName(ctx, "list-1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count) // no runs to cascade

	// List returns 2
	defs, err = store.ListDefinitions(ctx)
	require.NoError(t, err)
	assert.Len(t, defs, 2)

	// Delete non-existent is no-op (should not error)
	count, err = store.DeleteDefinitionByName(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestPipelineStore_GetStepRunsByRunID(t *testing.T) {
	client := getTestClient(t)
	store := NewPipelineStore(client)

	ctx := context.Background()
	run, err := store.CreateRun(ctx, "step-test-pipeline", "ev-step-001", "test.event")
	require.NoError(t, err)

	_, err = store.CreateStepRun(ctx, run.ID, "step-a", "notify", "send", map[string]any{"to": "user1"}, 1)
	require.NoError(t, err)
	_, err = store.CreateStepRun(ctx, run.ID, "step-b", "transform", "map", map[string]any{"key": "val"}, 1)
	require.NoError(t, err)
	_, err = store.CreateStepRun(ctx, run.ID, "step-c", "store", "save", map[string]any{}, 1)
	require.NoError(t, err)

	tests := []struct {
		name      string
		runID     int64
		wantCount int
	}{
		{
			name:      "happy path - returns all 3 step runs",
			runID:     run.ID,
			wantCount: 3,
		},
		{
			name:      "non-existent run returns empty list",
			runID:     run.ID + 99999,
			wantCount: 0,
		},
		{
			name:      "zero run ID returns empty list",
			runID:     0,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := store.GetStepRunsByRunID(ctx, tt.runID)
			require.NoError(t, err)
			assert.Len(t, steps, tt.wantCount)
			if tt.wantCount > 0 {
				assert.Equal(t, "step-a", steps[0].StepName)
				assert.Equal(t, "step-b", steps[1].StepName)
				assert.Equal(t, "step-c", steps[2].StepName)
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

// ---------------------------------------------------------------------------
// PageDataStore tests
// ---------------------------------------------------------------------------

func TestPageDataStore_CreateAndGet(t *testing.T) {
	client := getTestClient(t)
	store := NewPageDataStore(client)
	ctx := context.Background()

	// Seed a page for the retrieval test
	err := store.CreatePageData(ctx, "seed-token", "text", "Seed", types.KV{"content": "hello"}, "testuser", nil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		token    string
		wantNil  bool
		wantType string
		wantErr  bool
	}{
		{
			name:     "retrieve existing page",
			token:    "seed-token",
			wantNil:  false,
			wantType: "text",
			wantErr:  false,
		},
		{
			name:    "retrieve non-existent page",
			token:   "nonexistent-token",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "retrieve with empty token",
			token:   "",
			wantNil: true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageData, err := store.GetPageDataByToken(ctx, tt.token)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.wantNil {
				assert.Nil(t, pageData)
			} else {
				assert.NotNil(t, pageData)
				if !tt.wantNil {
					assert.Equal(t, tt.wantType, pageData.Type)
				}
			}
		})
	}
}

func TestPageDataStore_CreateDuplicateToken(t *testing.T) {
	client := getTestClient(t)
	store := NewPageDataStore(client)
	ctx := context.Background()

	token := "dup-token"
	err := store.CreatePageData(ctx, token, "text", "First", types.KV{"content": "a"}, "user1", nil)
	require.NoError(t, err)

	err = store.CreatePageData(ctx, token, "text", "Second", types.KV{"content": "b"}, "user2", nil)
	require.Error(t, err, "duplicate token should return error")
}

func TestPageDataStore_Delete(t *testing.T) {
	client := getTestClient(t)
	store := NewPageDataStore(client)
	ctx := context.Background()

	token := "del-token"
	err := store.CreatePageData(ctx, token, "text", "ToDelete", types.KV{}, "user", nil)
	require.NoError(t, err)

	tests := []struct {
		name         string
		token        string
		wantAffected int
		wantErr      bool
	}{
		{
			name:         "delete existing page",
			token:        token,
			wantAffected: 1,
			wantErr:      false,
		},
		{
			name:         "delete already deleted page",
			token:        token,
			wantAffected: 0,
			wantErr:      false,
		},
		{
			name:         "delete non-existent page",
			token:        "no-such-token",
			wantAffected: 0,
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affected, err := store.DeletePageData(ctx, tt.token)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantAffected, affected)
		})
	}
}

func TestPageDataStore_DeleteExpired(t *testing.T) {
	client := getTestClient(t)
	store := NewPageDataStore(client)
	ctx := context.Background()

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	oneHourLater := time.Now().Add(1 * time.Hour)

	err := store.CreatePageData(ctx, "expired-token", "text", "Expired", types.KV{}, "user", &oneHourAgo)
	require.NoError(t, err)

	err = store.CreatePageData(ctx, "active-token", "text", "Active", types.KV{}, "user", &oneHourLater)
	require.NoError(t, err)

	err = store.CreatePageData(ctx, "no-expiry-token", "text", "Forever", types.KV{}, "user", nil)
	require.NoError(t, err)

	count, err := store.DeleteExpiredPageData(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "only the expired page should be deleted")

	// Verify expired page is gone
	pageData, err := store.GetPageDataByToken(ctx, "expired-token")
	require.NoError(t, err)
	assert.Nil(t, pageData, "expired page should be deleted")

	// Verify active pages remain
	pageData, err = store.GetPageDataByToken(ctx, "active-token")
	require.NoError(t, err)
	assert.NotNil(t, pageData, "active page should remain")

	pageData, err = store.GetPageDataByToken(ctx, "no-expiry-token")
	require.NoError(t, err)
	assert.NotNil(t, pageData, "no-expiry page should remain")
}

// ---------------------------------------------------------------------------
// EventStore tests
// ---------------------------------------------------------------------------

func TestListDataEvents(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	store := NewEventStore(client)
	ctx := context.Background()

	events := []types.DataEvent{
		{EventID: "evt-001", EventType: "issue.created", Source: "github", Capability: "forge", EntityID: "repo#42"},
		{EventID: "evt-002", EventType: "bookmark.created", Source: "karakeep", Capability: "bookmark", EntityID: "url-1"},
		{EventID: "evt-003", EventType: "entry.new", Source: "reader", Capability: "reader", EntityID: "feed-5"},
	}

	for _, e := range events {
		require.NoError(t, store.AppendDataEvent(ctx, e))
	}

	tests := []struct {
		name          string
		opts          ListDataEventsOptions
		wantCount     int
		wantHasCursor bool
	}{
		{
			name:      "list all events",
			opts:      ListDataEventsOptions{Limit: 10},
			wantCount: 3,
		},
		{
			name:      "filter by source github",
			opts:      ListDataEventsOptions{Limit: 10, Source: "github"},
			wantCount: 1,
		},
		{
			name:      "filter by unknown source returns empty",
			opts:      ListDataEventsOptions{Limit: 10, Source: "unknown"},
			wantCount: 0,
		},
		{
			name:      "filter by event type",
			opts:      ListDataEventsOptions{Limit: 10, EventType: "bookmark.created"},
			wantCount: 1,
		},
		{
			name:          "pagination with cursor returns cursor",
			opts:          ListDataEventsOptions{Limit: 1},
			wantCount:     1,
			wantHasCursor: true,
		},
		{
			name:          "pagination last page no cursor",
			opts:          ListDataEventsOptions{Limit: 10},
			wantCount:     3,
			wantHasCursor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, cursor, err := store.ListDataEvents(ctx, tt.opts)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantCount)
			if tt.wantHasCursor {
				assert.NotEmpty(t, cursor)
			} else {
				assert.Empty(t, cursor)
			}
		})
	}
}
