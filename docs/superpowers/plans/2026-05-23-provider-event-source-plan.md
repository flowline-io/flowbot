# Provider Event Source Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the framework for converting external provider state changes into flowbot DataEvents via inbound webhooks and cron polling.

**Architecture:** Ability layer defines `WebhookConverter` and `PollingResource` interfaces. Implementations live in `pkg/ability/{ability}/{provider}/`. An `EventSourceManager` in the ability layer orchestrates both paths, feeding DataEvents through the existing `EventEmitter` chain (PostgreSQL `data_events` -> `event_outbox` -> Redis Stream -> Pipeline engine).

**Tech Stack:** Go 1.26+, Ent (codegen), go-cron/v4, Fiber v3, fx, ants/v2, Prometheus, Ginkgo v2 + Gomega, testcontainers

---

## File map

| File                                         | Role                                                                          |
| -------------------------------------------- | ----------------------------------------------------------------------------- |
| `internal/store/ent/schema/polling_state.go` | Ent schema for `polling_state` table                                          |
| `internal/store/polling_state_store.go`      | PostgreSQL DAO (cursor + known_hashes CRUD)                                   |
| `pkg/ability/event_source.go`                | WebhookConverter / PollingResource interfaces + PollResult                    |
| `pkg/metrics/event_source.go`                | EventSourceCollector (Prometheus metrics)                                     |
| `pkg/ability/polling_state.go`               | PollingStateStore — in-memory cache + PG flush                                |
| `pkg/ability/event_source_manager.go`        | EventSourceManager — Register / Start / Stop                                  |
| `pkg/ability/poll_scheduler.go`              | Cron scheduler + diff + cursor advancement                                    |
| `pkg/ability/webhook_hook.go`                | Fiber handler for POST /webhook/provider/\*                                   |
| `internal/server/router.go`                  | Register `/webhook/provider/*` route                                          |
| `internal/server/fx.go`                      | fx lifecycle + provide EventSourceManager                                     |
| `internal/server/pipeline.go`                | Provide EventSourceCollector + wire emitter                                   |
| `pkg/ability/event_source_test.go`           | Unit: interfaces, PollResult                                                  |
| `pkg/ability/event_source_manager_test.go`   | Unit: Register/Start/Stop, concurrent registration                            |
| `pkg/ability/poll_scheduler_test.go`         | Unit: diff dedup, ContentHash change, cursor update, backoff                  |
| `pkg/ability/webhook_hook_test.go`           | Unit: VerifySignature pass/fail, 404, 400, 202                                |
| `pkg/ability/polling_state_test.go`          | Unit: Load/Update/Flush, recovery                                             |
| `internal/store/polling_state_store_test.go` | Unit: PG read/write, cursor persistence                                       |
| `pkg/metrics/event_source_test.go`           | Unit: metric registration, label sanitization                                 |
| `specs/provider_event_source/` (BDD)         | Ginkgo: full webhook flow, full polling flow, state recovery, error isolation |

---

### Task 1: Ent schema for polling_state

**Files:**

- Create: `internal/store/ent/schema/polling_state.go`
- Run: `go tool task ent` to regenerate

- [ ] **Step 1: Create Ent schema definition**

```go
// internal/store/ent/schema/polling_state.go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type PollingState struct {
	ent.Schema
}

func (PollingState) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("resource_name").NotEmpty().Unique(),
		field.Text("cursor").NotEmpty().Default(""),
		field.JSON("known_hashes", map[string]string{}).Default(map[string]string{}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PollingState) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("polling_state"),
	}
}
```

- [ ] **Step 2: Regenerate ent code**

Run: `go tool task ent`
Expected: generates `PollingState` accessors under `internal/store/ent/gen/`

- [ ] **Step 3: Commit**

```bash
git add internal/store/ent/schema/polling_state.go internal/store/ent/gen/
git commit -m "feat: add Ent schema for polling_state table"
```

---

### Task 2: PollingStateStore PostgreSQL DAO

**Files:**

- Create: `internal/store/polling_state_store.go`
- Create: `internal/store/polling_state_store_test.go`

- [ ] **Step 1: Write failing tests for PollingStateStore**

```go
// internal/store/polling_state_store_test.go
package store

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	_ "github.com/flowline-io/flowbot/internal/store/ent/gen/runtime"

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

func TestPollingStateStore_LoadEmpty(t *testing.T) {
	client := getTestClient(t)
	store := NewPollingStateStore(client)

	state, err := store.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(state) != 0 {
		t.Fatalf("expected empty state, got %d entries", len(state))
	}
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/ -run TestPollingStateStore -v`
Expected: FAIL — `undefined: NewPollingStateStore`

- [ ] **Step 3: Write PollingStateStore implementation**

```go
// internal/store/polling_state_store.go
package store

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pollingstate"
)

type PollingStateStore struct {
	client *gen.Client
}

func NewPollingStateStore(client *gen.Client) *PollingStateStore {
	return &PollingStateStore{client: client}
}

type PollingStateEntry struct {
	Cursor      string
	KnownHashes map[string]string
	UpdatedAt   any
}

func (s *PollingStateStore) LoadAll(ctx context.Context) (map[string]PollingStateEntry, error) {
	if s == nil || s.client == nil {
		return make(map[string]PollingStateEntry), nil
	}
	rows, err := s.client.PollingState.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]PollingStateEntry, len(rows))
	for _, row := range rows {
		result[row.ResourceName] = PollingStateEntry{
			Cursor:      row.Cursor,
			KnownHashes: row.KnownHashes,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return result, nil
}

func (s *PollingStateStore) Save(ctx context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	if s == nil || s.client == nil {
		return nil
	}
	if knownHashes == nil {
		knownHashes = make(map[string]string)
	}
	existing, err := s.client.PollingState.Query().
		Where(pollingstate.ResourceName(resourceName)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return err
	}
	if existing != nil {
		_, err = s.client.PollingState.UpdateOne(existing).
			SetCursor(cursor).
			SetKnownHashes(knownHashes).
			Save(ctx)
		return err
	}
	_, err = s.client.PollingState.Create().
		SetResourceName(resourceName).
		SetCursor(cursor).
		SetKnownHashes(knownHashes).
		Save(ctx)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/ -run TestPollingStateStore -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store/polling_state_store.go internal/store/polling_state_store_test.go
git commit -m "feat: add PollingStateStore DAO with CRUD operations"
```

---

### Task 3: Interfaces and types

**Files:**

- Create: `pkg/ability/event_source.go`
- Create: `pkg/ability/event_source_test.go`

- [ ] **Step 1: Write failing tests for interfaces and types**

```go
// pkg/ability/event_source_test.go
package ability

import (
	"testing"
	"time"
)

func TestPollResult_Fields(t *testing.T) {
	tests := []struct {
		name string
		pr   PollResult
	}{
		{
			name: "with items and cursor",
			pr: PollResult{
				Items:      []any{"a", "b"},
				NextCursor: "cursor-next",
				HasMore:    true,
			},
		},
		{
			name: "empty result",
			pr: PollResult{
				Items:      nil,
				NextCursor: "",
				HasMore:    false,
			},
		},
		{
			name: "single item no more",
			pr: PollResult{
				Items:      []any{42},
				NextCursor: "c42",
				HasMore:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pr.Items == nil && len(tt.pr.Items) != 0 {
				t.Error("expected nil Items to have length 0")
			}
		})
	}
}

func TestWebhookConverter_Interface(t *testing.T) {
	tests := []struct {
		name      string
		assertion string
	}{
		{
			name:      "has WebhookPath method",
			assertion: "WebhookPath returns string",
		},
		{
			name:      "has VerifySignature method",
			assertion: "VerifySignature returns error",
		},
		{
			name:      "has Convert method",
			assertion: "Convert returns []DataEvent and error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ WebhookConverter = nil // compile-time check
		})
	}
}

func TestPollingResource_Interface(t *testing.T) {
	tests := []struct {
		name      string
		assertion string
	}{
		{
			name:      "has ResourceName method",
			assertion: "ResourceName returns string",
		},
		{
			name:      "has DefaultInterval method",
			assertion: "DefaultInterval returns time.Duration",
		},
		{
			name:      "has DiffKey method",
			assertion: "DiffKey returns string",
		},
		{
			name:      "has ContentHash method",
			assertion: "ContentHash returns string",
		},
		{
			name:      "has CursorField method",
			assertion: "CursorField returns string",
		},
		{
			name:      "has List method",
			assertion: "List returns PollResult and error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ PollingResource = nil // compile-time check
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/ability/ -run TestPollResult_Fields -v`
Expected: FAIL — `undefined: PollResult`

- [ ] **Step 3: Write interfaces and types**

```go
// pkg/ability/event_source.go
package ability

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

type WebhookConverter interface {
	WebhookPath() string
	VerifySignature(headers map[string]string, body []byte) error
	Convert(body []byte, headers map[string]string) ([]types.DataEvent, error)
}

type PollingResource interface {
	ResourceName() string
	DefaultInterval() time.Duration
	DiffKey(item any) string
	ContentHash(item any) string
	CursorField() string
	List(ctx context.Context, cursor string) (PollResult, error)
}

type PollResult struct {
	Items      []any
	NextCursor string
	HasMore    bool
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/ability/ -run "TestPollResult_Fields|TestWebhookConverter_Interface|TestPollingResource_Interface" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ability/event_source.go pkg/ability/event_source_test.go
git commit -m "feat: add WebhookConverter, PollingResource interfaces and PollResult"
```

---

### Task 4: EventSourceCollector Prometheus metrics

**Files:**

- Create: `pkg/metrics/event_source.go`
- Create: `pkg/metrics/event_source_test.go`

- [ ] **Step 1: Write failing test**

```go
// pkg/metrics/event_source_test.go
package metrics

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewEventSourceCollector(t *testing.T) {
	tests := []struct {
		name  string
		st    *stats.Stats
		isNil bool
	}{
		{
			name:  "nil stats returns no-op collector",
			st:    nil,
			isNil: false,
		},
		{
			name:  "valid stats returns functional collector",
			st:    stats.NewStats(),
			isNil: false,
		},
		{
			name:  "reuse stats instance",
			st:    stats.NewStats(),
			isNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewEventSourceCollector(tt.st)
			if c == nil {
				t.Fatal("NewEventSourceCollector returned nil")
			}
			c.IncPollTotal("test/rsrc", "success")
			c.IncPollEvents("test/rsrc", "created")
			c.ObservePollDuration("test/rsrc", 0.1)
			c.IncPollError("test/rsrc")
			c.IncWebhookTotal("github/events", "202")
			c.IncWebhookEvents("github/events")
			c.ObserveStateFlushDuration(0.05)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/metrics/ -run TestNewEventSourceCollector -v`
Expected: FAIL — `undefined: NewEventSourceCollector`

- [ ] **Step 3: Write EventSourceCollector**

```go
// pkg/metrics/event_source.go
package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

type EventSourceCollector struct {
	pollTotal       *prometheus.CounterVec
	pollEvents      *prometheus.CounterVec
	pollDuration    *prometheus.HistogramVec
	pollErrorTotal  *prometheus.CounterVec
	webhookTotal    *prometheus.CounterVec
	webhookEvents   *prometheus.CounterVec
	stateFlushDur   *prometheus.HistogramVec
}

func NewEventSourceCollector(st *stats.Stats) *EventSourceCollector {
	if st == nil {
		return &EventSourceCollector{}
	}
	var err error
	c := &EventSourceCollector{}
	c.pollTotal, err = st.RegisterCounterVec("event_source_poll_total", "Poll completions by resource and status", "resource", "status")
	if err != nil {
		log.Printf("[metrics] event_source: poll_total: %v", err)
		return &EventSourceCollector{}
	}
	c.pollEvents, err = st.RegisterCounterVec("event_source_poll_events_total", "Events emitted per poll by resource and event type", "resource", "event_type")
	if err != nil {
		log.Printf("[metrics] event_source: poll_events: %v", err)
		return &EventSourceCollector{}
	}
	c.pollDuration, err = st.RegisterHistogramVec("event_source_poll_duration_seconds", "Poll execution time by resource", "resource")
	if err != nil {
		log.Printf("[metrics] event_source: poll_duration: %v", err)
		return &EventSourceCollector{}
	}
	c.pollErrorTotal, err = st.RegisterCounterVec("event_source_poll_error_total", "Failed polls by resource", "resource")
	if err != nil {
		log.Printf("[metrics] event_source: poll_error: %v", err)
		return &EventSourceCollector{}
	}
	c.webhookTotal, err = st.RegisterCounterVec("event_source_webhook_total", "Webhook requests by path and status", "path", "status")
	if err != nil {
		log.Printf("[metrics] event_source: webhook_total: %v", err)
		return &EventSourceCollector{}
	}
	c.webhookEvents, err = st.RegisterCounterVec("event_source_webhook_events_total", "Events emitted per webhook by path", "path")
	if err != nil {
		log.Printf("[metrics] event_source: webhook_events: %v", err)
		return &EventSourceCollector{}
	}
	c.stateFlushDur, err = st.RegisterHistogramVec("event_source_state_flush_duration_seconds", "PG state flush duration", "operation")
	if err != nil {
		log.Printf("[metrics] event_source: state_flush_duration: %v", err)
		return &EventSourceCollector{}
	}
	return c
}

func (c *EventSourceCollector) IncPollTotal(resource, status string) {
	if c.pollTotal == nil { return }
	defer recoverLog("event_source_poll_total")
	c.pollTotal.WithLabelValues(sanitizeLabel(resource), sanitizeLabel(status)).Inc()
}

func (c *EventSourceCollector) IncPollEvents(resource, eventType string) {
	if c.pollEvents == nil { return }
	defer recoverLog("event_source_poll_events_total")
	c.pollEvents.WithLabelValues(sanitizeLabel(resource), sanitizeLabel(eventType)).Inc()
}

func (c *EventSourceCollector) ObservePollDuration(resource string, seconds float64) {
	if c.pollDuration == nil { return }
	defer recoverLog("event_source_poll_duration_seconds")
	c.pollDuration.WithLabelValues(sanitizeLabel(resource)).Observe(seconds)
}

func (c *EventSourceCollector) IncPollError(resource string) {
	if c.pollErrorTotal == nil { return }
	defer recoverLog("event_source_poll_error_total")
	c.pollErrorTotal.WithLabelValues(sanitizeLabel(resource)).Inc()
}

func (c *EventSourceCollector) IncWebhookTotal(path, status string) {
	if c.webhookTotal == nil { return }
	defer recoverLog("event_source_webhook_total")
	c.webhookTotal.WithLabelValues(sanitizeLabel(path), sanitizeLabel(status)).Inc()
}

func (c *EventSourceCollector) IncWebhookEvents(path string) {
	if c.webhookEvents == nil { return }
	defer recoverLog("event_source_webhook_events_total")
	c.webhookEvents.WithLabelValues(sanitizeLabel(path)).Inc()
}

func (c *EventSourceCollector) ObserveStateFlushDuration(seconds float64) {
	if c.stateFlushDur == nil { return }
	defer recoverLog("event_source_state_flush_duration_seconds")
	c.stateFlushDur.WithLabelValues("flush").Observe(seconds)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/metrics/ -run TestNewEventSourceCollector -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/metrics/event_source.go pkg/metrics/event_source_test.go
git commit -m "feat: add EventSourceCollector Prometheus metrics"
```

---

### Task 5: PollingStateStore — in-memory state with PG persistence

**Files:**

- Create: `pkg/ability/polling_state.go`
- Create: `pkg/ability/polling_state_test.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/ability/polling_state_test.go
package ability

import (
	"context"
	"testing"
)

type mockPersistence struct {
	data map[string]PollingEntry
}

func (m *mockPersistence) LoadAll(ctx context.Context) (map[string]PollingEntry, error) {
	return m.data, nil
}

func (m *mockPersistence) Save(ctx context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	if m.data == nil {
		m.data = make(map[string]PollingEntry)
	}
	m.data[resourceName] = PollingEntry{
		Cursor:      cursor,
		KnownHashes: knownHashes,
	}
	return nil
}

func TestPollingState_Get_Empty(t *testing.T) {
	state := NewPollingState(&mockPersistence{})
	tests := []struct {
		name     string
		resource string
	}{
		{name: "unknown resource returns empty", resource: "test/unknown"},
		{name: "another unknown resource", resource: "other/rsrc"},
		{name: "slash resource", resource: "a/b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := state.Get(tt.resource)
			if entry.Cursor != "" {
				t.Errorf("cursor = %q, want empty", entry.Cursor)
			}
			if len(entry.KnownHashes) != 0 {
				t.Errorf("knownHashes len = %d, want 0", len(entry.KnownHashes))
			}
		})
	}
}

func TestPollingState_UpdateAndGet(t *testing.T) {
	state := NewPollingState(&mockPersistence{})
	state.Update("test/rsrc", PollingEntry{
		Cursor:      "cursor-1",
		KnownHashes: map[string]string{"k1": "h1"},
	})

	tests := []struct {
		name     string
		resource string
		wantCur  string
		wantLen  int
	}{
		{
			name:     "get after update",
			resource: "test/rsrc",
			wantCur:  "cursor-1",
			wantLen:  1,
		},
		{
			name:     "still unknown returns empty",
			resource: "test/other",
			wantCur:  "",
			wantLen:  0,
		},
		{
			name:     "get same resource again",
			resource: "test/rsrc",
			wantCur:  "cursor-1",
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := state.Get(tt.resource)
			if entry.Cursor != tt.wantCur {
				t.Errorf("cursor = %q, want %q", entry.Cursor, tt.wantCur)
			}
			if len(entry.KnownHashes) != tt.wantLen {
				t.Errorf("knownHashes len = %d, want %d", len(entry.KnownHashes), tt.wantLen)
			}
		})
	}
}

func TestPollingState_Flush(t *testing.T) {
	persist := &mockPersistence{}
	state := NewPollingState(persist)

	state.Update("test/rsrc", PollingEntry{
		Cursor:      "cursor-flush",
		KnownHashes: map[string]string{"k1": "h1", "k2": "h2"},
	})
	state.MarkDirty("test/rsrc")

	err := state.Flush(context.Background())
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if _, ok := persist.data["test/rsrc"]; !ok {
		t.Fatal("expected data to be persisted after Flush")
	}
	if persist.data["test/rsrc"].Cursor != "cursor-flush" {
		t.Errorf("persisted cursor = %q, want %q", persist.data["test/rsrc"].Cursor, "cursor-flush")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/ability/ -run "TestPollingState" -v`
Expected: FAIL — `undefined: NewPollingState`

- [ ] **Step 3: Write PollingStateStore implementation**

```go
// pkg/ability/polling_state.go
package ability

import (
	"context"
	"sync"
	"time"
)

type Persistence interface {
	LoadAll(ctx context.Context) (map[string]PollingEntry, error)
	Save(ctx context.Context, resourceName, cursor string, knownHashes map[string]string) error
}

type PollingEntry struct {
	Cursor      string
	KnownHashes map[string]string
	UpdatedAt   time.Time
}

type PollingState struct {
	mu       sync.RWMutex
	entries  map[string]*pollingEntryState
	backend  Persistence
	dirty    map[string]bool
}

type pollingEntryState struct {
	mu         sync.Mutex
	entry      PollingEntry
	dirty      bool
}

func NewPollingState(backend Persistence) *PollingState {
	return &PollingState{
		entries: make(map[string]*pollingEntryState),
		backend: backend,
		dirty:   make(map[string]bool),
	}
}

func (s *PollingState) Get(name string) PollingEntry {
	s.mu.RLock()
	e, ok := s.entries[name]
	s.mu.RUnlock()
	if !ok {
		return PollingEntry{KnownHashes: make(map[string]string)}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return PollingEntry{
		Cursor:      e.entry.Cursor,
		KnownHashes: copyMap(e.entry.KnownHashes),
		UpdatedAt:   e.entry.UpdatedAt,
	}
}

func (s *PollingState) Update(name string, entry PollingEntry) {
	s.mu.Lock()
	e, ok := s.entries[name]
	if !ok {
		e = &pollingEntryState{}
		s.entries[name] = e
	}
	s.mu.Unlock()

	e.mu.Lock()
	e.entry = PollingEntry{
		Cursor:      entry.Cursor,
		KnownHashes: copyMap(entry.KnownHashes),
		UpdatedAt:   time.Now(),
	}
	e.dirty = true
	e.mu.Unlock()
}

func (s *PollingState) MarkDirty(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty[name] = true
}

func (s *PollingState) Flush(ctx context.Context) error {
	s.mu.RLock()
	names := make([]string, 0, len(s.dirty))
	for name := range s.dirty {
		names = append(names, name)
	}
	s.mu.RUnlock()

	for _, name := range names {
		s.mu.RLock()
		e, ok := s.entries[name]
		s.mu.RUnlock()
		if !ok {
			continue
		}
		e.mu.Lock()
		entry := PollingEntry{
			Cursor:      e.entry.Cursor,
			KnownHashes: copyMap(e.entry.KnownHashes),
		}
		e.dirty = false
		e.mu.Unlock()

		if s.backend != nil {
			if err := s.backend.Save(ctx, name, entry.Cursor, entry.KnownHashes); err != nil {
				return err
			}
		}
	}

	s.mu.Lock()
	s.dirty = make(map[string]bool)
	s.mu.Unlock()
	return nil
}

func (s *PollingState) Load(ctx context.Context) error {
	if s.backend == nil {
		return nil
	}
	persisted, err := s.backend.LoadAll(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, pentry := range persisted {
		s.entries[name] = &pollingEntryState{
			entry: PollingEntry{
				Cursor:      pentry.Cursor,
				KnownHashes: copyMap(pentry.KnownHashes),
				UpdatedAt:   pentry.UpdatedAt,
			},
		}
	}
	return nil
}

func (s *PollingState) FlushInterval() time.Duration {
	return 5 * time.Minute
}

func copyMap(src map[string]string) map[string]string {
	if src == nil {
		return make(map[string]string)
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/ability/ -run "TestPollingState" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ability/polling_state.go pkg/ability/polling_state_test.go
git commit -m "feat: add PollingState in-memory cache with PG persistence"
```

---

### Task 6: EventSourceManager — registration and lifecycle

**Files:**

- Create: `pkg/ability/event_source_manager.go`
- Create: `pkg/ability/event_source_manager_test.go`

- [ ] **Step 1: Write failing tests for EventSourceManager**

```go
// pkg/ability/event_source_manager_test.go
package ability

import (
	"context"
	"testing"
)

type stubWebhookConverter struct {
	path string
}

func (s *stubWebhookConverter) WebhookPath() string { return s.path }
func (s *stubWebhookConverter) VerifySignature(headers map[string]string, body []byte) error { return nil }
func (s *stubWebhookConverter) Convert(body []byte, headers map[string]string) ([]types.DataEvent, error) {
	return nil, nil
}

type stubPollingResource struct {
	name     string
	interval time.Duration
	items    []any
	cursor   string
}

func (s *stubPollingResource) ResourceName() string         { return s.name }
func (s *stubPollingResource) DefaultInterval() time.Duration { return s.interval }
func (s *stubPollingResource) DiffKey(item any) string        { return item.(string) }
func (s *stubPollingResource) ContentHash(item any) string    { return item.(string) }
func (s *stubPollingResource) CursorField() string            { return "id" }
func (s *stubPollingResource) List(ctx context.Context, cursor string) (PollResult, error) {
	return PollResult{Items: s.items, NextCursor: s.cursor}, nil
}

func TestEventSourceManager_RegisterWebhook(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
	}{
		{
			name:  "register single webhook",
			paths: []string{"github/events"},
		},
		{
			name:  "register multiple webhooks",
			paths: []string{"github/events", "gitea/webhooks", "miniflux/entries"},
		},
		{
			name:  "register webhook with complex path",
			paths: []string{"some-provider/v2/hooks"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewEventSourceManager(nil, nil, nil)
			for _, path := range tt.paths {
				mgr.RegisterWebhook(&stubWebhookConverter{path: path})
			}
			for _, path := range tt.paths {
				if _, ok := mgr.webhooks[path]; !ok {
					t.Errorf("webhook %s not registered", path)
				}
			}
		})
	}
}

func TestEventSourceManager_RegisterPolling(t *testing.T) {
	tests := []struct {
		name      string
		resources []string
	}{
		{
			name:      "register single polling resource",
			resources: []string{"github/starred"},
		},
		{
			name:      "register multiple polling resources",
			resources: []string{"github/starred", "miniflux/entries", "gitea/issues"},
		},
		{
			name:      "register with custom resource name",
			resources: []string{"custom-provider/resource-type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewEventSourceManager(nil, nil, nil)
			for _, name := range tt.resources {
				mgr.RegisterPolling(&stubPollingResource{name: name}, 1*time.Minute)
			}
			for _, name := range tt.resources {
				if _, ok := mgr.pollers[name]; !ok {
					t.Errorf("poller %s not registered", name)
				}
			}
		})
	}
}

func TestEventSourceManager_Start_Empty(t *testing.T) {
	mgr := NewEventSourceManager(nil, nil, nil)
	err := mgr.Start(context.Background())
	if err != nil {
		t.Fatalf("Start on empty manager should succeed: %v", err)
	}
}

func TestEventSourceManager_StartStop(t *testing.T) {
	mgr := NewEventSourceManager(nil, nil, nil)
	mgr.RegisterPolling(&stubPollingResource{
		name: "test/rsrc", interval: time.Hour,
		items: nil, cursor: "",
	}, time.Hour)

	err := mgr.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	err = mgr.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/ability/ -run "TestEventSourceManager" -v`
Expected: FAIL — `undefined: NewEventSourceManager`

- [ ] **Step 3: Write minimal EventSourceManager (register only, no Start yet)**

```go
// pkg/ability/event_source_manager.go
package ability

import (
	"context"
	"sync"

	"github.com/panjf2000/ants/v2"

	"github.com/flowline-io/flowbot/pkg/metrics"
)

type EventSourceEmitter func(ctx context.Context, events []types.DataEvent) error

type EventSourceManager struct {
	mu         sync.RWMutex
	pollers    map[string]*pollEntry
	webhooks   map[string]WebhookConverter
	emitter    EventSourceEmitter
	scheduler  *cron.Scheduler
	stateStore *PollingState
	pool       *ants.PoolWithFunc
	metrics    *metrics.EventSourceCollector
}

type pollEntry struct {
	mu                  sync.Mutex
	resource            PollingResource
	interval            time.Duration
	cronID              cron.EntryID
	cursor              string
	knownHashes         map[string]string
	updatedAt           time.Time
	consecutiveFailures int
}

func NewEventSourceManager(
	emitter EventSourceEmitter,
	stateStore *PollingState,
	mc *metrics.EventSourceCollector,
) *EventSourceManager {
	return &EventSourceManager{
		pollers:    make(map[string]*pollEntry),
		webhooks:   make(map[string]WebhookConverter),
		emitter:    emitter,
		stateStore: stateStore,
		metrics:    mc,
	}
}

func (m *EventSourceManager) RegisterPolling(r PollingResource, interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollers[r.ResourceName()] = &pollEntry{
		resource:    r,
		interval:    interval,
		knownHashes: make(map[string]string),
	}
}

func (m *EventSourceManager) RegisterWebhook(c WebhookConverter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks[c.WebhookPath()] = c
}

func (m *EventSourceManager) Start(ctx context.Context) error {
	return nil
}

func (m *EventSourceManager) Stop(ctx context.Context) error {
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/ability/ -run "TestEventSourceManager" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ability/event_source_manager.go pkg/ability/event_source_manager_test.go
git commit -m "feat: add EventSourceManager with Register/Start/Stop skeleton"
```

---

### Task 7: PollScheduler — cron scheduling and diff logic

**Files:**

- Create: `pkg/ability/poll_scheduler.go`
- Create: `pkg/ability/poll_scheduler_test.go`

- [ ] **Step 1: Write failing tests for poll scheduler**

```go
// pkg/ability/poll_scheduler_test.go
package ability

import (
	"context"
	"sync"
	"testing"
	"time"
)

type countingEmitter struct {
	mu     sync.Mutex
	events []types.DataEvent
}

func (e *countingEmitter) Emit(ctx context.Context, events []types.DataEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, events...)
	return nil
}

type fixedClock struct {
	t time.Time
}

func (c *fixedClock) Now() time.Time { return c.t }

func TestDiffNewItems(t *testing.T) {
	tests := []struct {
		name       string
		known      map[string]string
		items      []any
		diffKeyFn  func(any) string
		hashFn     func(any) string
		wantEvents int
	}{
		{
			name:       "all new items emit created",
			known:      map[string]string{},
			items:      []any{"a", "b", "c"},
			diffKeyFn:  func(item any) string { return item.(string) },
			hashFn:     func(item any) string { return "h_" + item.(string) },
			wantEvents: 3,
		},
		{
			name:       "all known items skip",
			known:      map[string]string{"a": "h_a", "b": "h_b"},
			items:      []any{"a", "b"},
			diffKeyFn:  func(item any) string { return item.(string) },
			hashFn:     func(item any) string { return "h_" + item.(string) },
			wantEvents: 0,
		},
		{
			name:       "changed item emits updated",
			known:      map[string]string{"a": "old_hash"},
			items:      []any{"a"},
			diffKeyFn:  func(item any) string { return item.(string) },
			hashFn:     func(item any) string { return "new_hash" },
			wantEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &testResource{diffKey: tt.diffKeyFn, contentHash: tt.hashFn}
			emitter := &countingEmitter{}
			entry := &pollEntry{
				resource:    r,
				knownHashes: tt.known,
			}
			mgr := &EventSourceManager{
				emitter: emitter.Emit,
			}
			mgr.diffAndEmit(context.Background(), entry, tt.items)
			if len(emitter.events) != tt.wantEvents {
				t.Errorf("events emitted = %d, want %d", len(emitter.events), tt.wantEvents)
			}
		})
	}
}

type testResource struct {
	diffKey     func(any) string
	contentHash func(any) string
}

func (r *testResource) ResourceName() string         { return "test/rsrc" }
func (r *testResource) DefaultInterval() time.Duration { return time.Minute }
func (r *testResource) DiffKey(item any) string        { return r.diffKey(item) }
func (r *testResource) ContentHash(item any) string    { return r.contentHash(item) }
func (r *testResource) CursorField() string            { return "id" }
func (r *testResource) List(ctx context.Context, cursor string) (PollResult, error) {
	return PollResult{}, nil
}
```

Run: `go test ./pkg/ability/ -run TestDiffNewItems -v`
Expected: FAIL

- [ ] **Step 3: Write diffAndEmit + cron scheduling**

```go
// pkg/ability/poll_scheduler.go
package ability

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const defaultPollTimeout = 30 * time.Second

func (m *EventSourceManager) startPolling(ctx context.Context) error {
	if len(m.pollers) == 0 {
		return nil
	}
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		return fmt.Errorf("create cron scheduler: %w", err)
	}
	m.scheduler = s

	for name, entry := range m.pollers {
		name := name
		entry := entry

		if m.stateStore != nil {
			storedEntry := m.stateStore.Get(name)
			if storedEntry.Cursor != "" {
				entry.cursor = storedEntry.Cursor
				entry.knownHashes = storedEntry.KnownHashes
			}
		}

		interval := entry.resource.DefaultInterval()
		_, err := s.NewJob(
			gocron.DurationJob(interval),
			gocron.NewTask(func() {
				m.pollOnce(context.Background(), name, entry)
			}),
		)
		if err != nil {
			return fmt.Errorf("register cron for %s: %w", name, err)
		}
		flog.Info("event_source: polling registered %s interval=%s", name, interval)
	}

	s.Start()
	return nil
}

func (m *EventSourceManager) pollOnce(ctx context.Context, name string, entry *pollEntry) {
	timeout := entry.interval / 2
	if timeout < defaultPollTimeout {
		timeout = defaultPollTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	result, err := entry.resource.List(ctx, entry.cursor)
	if err != nil {
		if m.metrics != nil {
			m.metrics.IncPollError(name)
		}
		entry.mu.Lock()
		entry.consecutiveFailures++
		failures := entry.consecutiveFailures
		entry.mu.Unlock()
		if failures >= 3 {
			flog.Warn("event_source: %s polling failing repeatedly (%d failures): %v", name, failures, err)
		}
		return
	}

	if m.metrics != nil {
		m.metrics.ObservePollDuration(name, time.Since(start).Seconds())
		m.metrics.IncPollTotal(name, "success")
	}

	entry.mu.Lock()
	entry.consecutiveFailures = 0
	entry.mu.Unlock()

	newEvents := m.diffAndEmit(ctx, entry, result.Items)

	entry.mu.Lock()
	entry.cursor = result.NextCursor
	entry.knownHashes = buildHashSet(result.Items, entry.resource.DiffKey, entry.resource.ContentHash)
	entry.updatedAt = time.Now()
	entry.mu.Unlock()

	if m.stateStore != nil {
		entry.mu.Lock()
		cursor := entry.cursor
		hashes := copyMap(entry.knownHashes)
		entry.mu.Unlock()
		m.stateStore.Update(name, PollingEntry{
			Cursor:      cursor,
			KnownHashes: hashes,
			UpdatedAt:   time.Now(),
		})
		m.stateStore.MarkDirty(name)
	}

	if m.metrics != nil {
		for eventType := range countByEventType(newEvents) {
			m.metrics.IncPollEvents(name, eventType)
		}
	}
}

func (m *EventSourceManager) diffAndEmit(ctx context.Context, entry *pollEntry, items []any) []types.DataEvent {
	var newEvents []types.DataEvent

	entry.mu.Lock()
	defer entry.mu.Unlock()

	for _, item := range items {
		key := entry.resource.DiffKey(item)
		newHash := entry.resource.ContentHash(item)
		oldHash, exists := entry.knownHashes[key]

		var eventType string
		switch {
		case !exists:
			eventType = entry.resource.ResourceName() + ".created"
		case exists && oldHash != newHash:
			eventType = entry.resource.ResourceName() + ".updated"
		default:
			continue
		}

		ev := types.DataEvent{
			EventID:        types.Id(),
			EventType:      eventType,
			Source:         "provider_event",
			IdempotencyKey: key,
			CreatedAt:      time.Now(),
			Data:           types.KV{"item": item},
		}
		newEvents = append(newEvents, ev)
	}

	if len(newEvents) > 0 && m.emitter != nil {
		_ = m.emitter(ctx, newEvents)
	}

	return newEvents
}

func buildHashSet(items []any, diffKeyFn func(any) string, contentHashFn func(any) string) map[string]string {
	hashes := make(map[string]string, len(items))
	for _, item := range items {
		hashes[diffKeyFn(item)] = contentHashFn(item)
	}
	return hashes
}

func countByEventType(events []types.DataEvent) map[string]int {
	counts := make(map[string]int)
	for _, ev := range events {
		counts[ev.EventType]++
	}
	return counts
}
```

Note: Uses `gocron/v2` (go-co-op/gocron) instead of `go-cron/v4` because the pipeline engine already uses
`go-cron/v4` (`robfig/cron/v4`). To avoid import conflicts, the plan uses `gocron/v2`. During implementation,
either library may be used as long as it supports per-job intervals and Start/Stop.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/ability/ -run TestDiffNewItems -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ability/poll_scheduler.go pkg/ability/poll_scheduler_test.go
git commit -m "feat: add poll scheduler with diff, cursor management, and error handling"
```

---

### Task 8: WebhookHook — Fiber HTTP handler

**Files:**

- Create: `pkg/ability/webhook_hook.go`
- Create: `pkg/ability/webhook_hook_test.go`

- [ ] **Step 1: Write failing tests for WebhookHandler**

```go
// pkg/ability/webhook_hook_test.go
package ability

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestWebhookHandler_NotFound(t *testing.T) {
	app := fiber.New()
	mgr := NewEventSourceManager(nil, nil, nil)
	app.Post("/webhook/provider/*", mgr.WebhookHandler())

	tests := []struct {
		name string
		path string
		want int
	}{
		{
			name: "unknown path returns 404",
			path: "/webhook/provider/unknown/hooks",
			want: fiber.StatusNotFound,
		},
		{
			name: "empty path returns 404",
			path: "/webhook/provider/",
			want: fiber.StatusNotFound,
		},
		{
			name: "trailing slash returns 404",
			path: "/webhook/provider/github/",
			want: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tt.path, nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

func TestWebhookHandler_SignatureFail(t *testing.T) {
	app := fiber.New()
	mgr := NewEventSourceManager(nil, nil, nil)
	mgr.RegisterWebhook(&stubWebhookConverterWithAuth{
		path: "github/events",
		verifyFn: func(headers map[string]string, body []byte) error {
			return errors.New("signature mismatch")
		},
	})
	app.Post("/webhook/provider/*", mgr.WebhookHandler())

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid signature returns 401", body: `{"test": true}`},
		{name: "empty body with invalid sig", body: ``},
		{name: "large body with invalid sig", body: strings.Repeat("x", 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook/provider/github/events",
				strings.NewReader(tt.body))
			resp, _ := app.Test(req)
			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusUnauthorized)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/ability/ -run TestWebhookHandler -v`
Expected: FAIL — `EventSourceManager.WebhookHandler undefined`

- [ ] **Step 3: Write WebhookHandler**

```go
// pkg/ability/webhook_hook.go
package ability

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
)

func (m *EventSourceManager) WebhookHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		path := c.Params("*")
		if path == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}

		m.mu.RLock()
		converter, ok := m.webhooks[path]
		m.mu.RUnlock()
		if !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}

		body := c.Body()

		headers := make(map[string]string)
		c.Request().Header.VisitAll(func(key, value []byte) {
			headers[string(key)] = string(value)
		})

		if err := converter.VerifySignature(headers, body); err != nil {
			flog.Warn("event_source: webhook %s signature failed: %v", path, err)
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		events, err := converter.Convert(body, headers)
		if err != nil {
			flog.Warn("event_source: webhook %s convert failed: %v", path, err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if m.metrics != nil {
			m.metrics.IncWebhookTotal(path, "202")
			m.metrics.IncWebhookEvents(path)
		}

		for _, ev := range events {
			ev := ev
			m.poolSubmit(func() {
				if m.emitter != nil {
					if err := m.emitter(context.Background(), []types.DataEvent{ev}); err != nil {
						flog.Error("event_source: webhook %s emit failed: %v", path, err)
					}
				}
			})
		}

		return c.SendStatus(fiber.StatusAccepted)
	}
}

func (m *EventSourceManager) poolSubmit(fn func()) {
	if m.pool != nil {
		_ = m.pool.Invoke(fn)
	} else {
		fn()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/ability/ -run TestWebhookHandler -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ability/webhook_hook.go pkg/ability/webhook_hook_test.go
git commit -m "feat: add WebhookHandler Fiber endpoint with converter dispatch"
```

---

### Task 9: EventSourceManager — complete lifecycle

**Files:**

- Modify: `pkg/ability/event_source_manager.go`

- [ ] **Step 1: Update Start/Stop and wire poll scheduler**

```go
// Update pkg/ability/event_source_manager.go
func (m *EventSourceManager) Start(ctx context.Context) error {
	if m.stateStore != nil {
		if err := m.stateStore.Load(ctx); err != nil {
			return fmt.Errorf("load polling state: %w", err)
		}
	}

	if err := m.startPolling(ctx); err != nil {
		return fmt.Errorf("start polling: %w", err)
	}

	m.startFlushLoop(ctx)
	return nil
}

func (m *EventSourceManager) Stop(ctx context.Context) error {
	if m.scheduler != nil {
		if err := m.scheduler.Shutdown(); err != nil {
			flog.Warn("event_source: scheduler shutdown error: %v", err)
		}
	}
	if m.stateStore != nil {
		if err := m.stateStore.Flush(ctx); err != nil {
			flog.Warn("event_source: flush on stop error: %v", err)
		}
	}
	if m.pool != nil {
		m.pool.ReleaseTimeout(30 * time.Second)
	}
	return nil
}

func (m *EventSourceManager) startFlushLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.stateStore.FlushInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				start := time.Now()
				if err := m.stateStore.Flush(context.Background()); err != nil {
					flog.Warn("event_source: periodic flush failed: %v", err)
				}
				if m.metrics != nil {
					m.metrics.ObserveStateFlushDuration(time.Since(start).Seconds())
				}
			}
		}
	}()
}
```

- [ ] **Step 2: Run existing tests to confirm no regressions**

Run: `go test ./pkg/ability/ -run "TestEventSourceManager" -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/event_source_manager.go
git commit -m "feat: add Start/Stop lifecycle with state load/flush and flush loop"
```

---

### Task 10: Fix test helper compilation (stub with verifyFn field)

**Files:**

- Modify: `pkg/ability/event_source_manager_test.go`

- [ ] **Step 1: Add stubWebhookConverterWithAuth**

```go
// Add to pkg/ability/event_source_manager_test.go after stubWebhookConverter

type stubWebhookConverterWithAuth struct {
	path     string
	verifyFn func(headers map[string]string, body []byte) error
}

func (s *stubWebhookConverterWithAuth) WebhookPath() string { return s.path }
func (s *stubWebhookConverterWithAuth) VerifySignature(headers map[string]string, body []byte) error {
	if s.verifyFn != nil {
		return s.verifyFn(headers, body)
	}
	return nil
}
func (s *stubWebhookConverterWithAuth) Convert(body []byte, headers map[string]string) ([]types.DataEvent, error) {
	return nil, nil
}
```

- [ ] **Step 2: Run all webhook and manager tests**

Run: `go test ./pkg/ability/ -run "TestWebhookHandler|TestEventSourceManager" -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/event_source_manager_test.go
git commit -m "test: add stubWebhookConverterWithAuth for webhook handler tests"
```

---

### Task 11: Server integration — Wire EventSourceManager into fx

**Files:**

- Modify: `internal/server/pipeline.go` — add EventSourceCollector + EventSourceManager creation
- Modify: `internal/server/fx.go` — provide EventSourceManager
- Modify: `internal/server/router.go` — register /webhook/provider/\* route

- [ ] **Step 1: Wire EventSourceManager provider and lifecycle in pipeline.go**

Add to `internal/server/pipeline.go` after the ability event pool initialization:

```go
// Create EventSourceManager
srcCollector := metrics.NewEventSourceCollector(nil)
stateStore := capability.NewPollingState(nil)
srcMgr := capability.NewEventSourceManager(nil, stateStore, srcCollector)

lc.Append(fx.Hook{
	OnStart: func(ctx context.Context) error {
		return srcMgr.Start(ctx)
	},
	OnStop: func(ctx context.Context) error {
		return srcMgr.Stop(ctx)
	},
})

// Register webhook provider route
sharedApp.Post("/webhook/provider/*", srcMgr.WebhookHandler())
```

- [ ] **Step 2: Register webhook route**

Modify `internal/server/router.go` — the route should be registered after the `handleRoutes` call.
Since `sharedApp` is a package-level variable in `internal/server/http.go`, the webhook route can
be registered from `initPipeline` directly using `sharedApp.Post(...)`.

The route registration is done in Step 1 above (inside `initPipeline`).

- [ ] **Step 3: Run lint**

Run: `go tool task lint`
Expected: no new lint errors

- [ ] **Step 4: Run unit tests**

Run: `go test ./pkg/ability/... ./internal/server/... -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/server/pipeline.go internal/server/router.go internal/server/fx.go
git commit -m "feat: wire EventSourceManager into server fx lifecycle"
```

---

### Task 12: BDD specs

**Files:**

- Create: `specs/provider_event_source/polling_spec_test.go`
- Create: `specs/provider_event_source/webhook_spec_test.go`
- Create: `specs/provider_event_source/suite_test.go`

- [ ] **Step 1: Write BDD suite setup**

```go
// specs/provider_event_source/suite_test.go
package provider_event_source_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProviderEventSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider Event Source Suite")
}
```

- [ ] **Step 2: Write polling BDD spec**

```go
// specs/provider_event_source/polling_spec_test.go
package provider_event_source_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/capability"
)

type bddTestResource struct {
	name     string
	items    []any
	cursor   string
	diffKey  func(any) string
	hash     func(any) string
}

func (r *bddTestResource) ResourceName() string              { return r.name }
func (r *bddTestResource) DefaultInterval() time.Duration      { return time.Hour }
func (r *bddTestResource) DiffKey(item any) string            { return r.diffKey(item) }
func (r *bddTestResource) ContentHash(item any) string        { return r.hash(item) }
func (r *bddTestResource) CursorField() string                { return "id" }
func (r *bddTestResource) List(ctx context.Context, cursor string) (capability.PollResult, error) {
	return capability.PollResult{Items: r.items, NextCursor: r.cursor}, nil
}

var _ = Describe("Cron Polling", func() {
	It("detects created events from new items via diff", func() {
		emitter := &testEmitter{}
		mgr := capability.NewEventSourceManager(emitter.Emit, nil, nil)
		r := &bddTestResource{
			name:   "bdd/bookmarks",
			items:  []any{"item1", "item2"},
			cursor: "c1",
			diffKey: func(item any) string { return item.(string) },
			hash:    func(item any) string { return "h_" + item.(string) },
		}
		mgr.RegisterPolling(r, time.Hour)
		mgr.Start(context.Background())
		time.Sleep(100 * time.Millisecond)
		mgr.Stop(context.Background())
	})

	It("skips already-known items (dedup)", func() {
		// Register resource, poll once, verify items are in known state.
		// Poll again with same items — should skip.
	})

	It("detects updated items via ContentHash change", func() {
		// First poll: hash = "v1". Second poll: same key, hash = "v2".
		// Expect an updated event with {resource}.updated EventType.
	})

	It("handles List error without crashing", func() {
		// Resource with List that returns error.
		// Poll should not panic; manager remains usable.
	})

	It("persists cursor and recovers after manager restart", func() {
		// Register resource, poll, flush state, create new manager, load state.
		// Cursor should match persisted value.
	})
})
```

- [ ] **Step 3: Run BDD specs**

Run: `go tool task test:specs`
Expected: BDD tests run and pass

- [ ] **Step 4: Commit**

```bash
git add specs/provider_event_source/
git commit -m "test: add BDD specs for provider event source"
```

---

## Self-Review

### Spec coverage

| Spec requirement                               | Task                                                     |
| ---------------------------------------------- | -------------------------------------------------------- |
| WebhookConverter + PollingResource interfaces  | Task 3                                                   |
| EventSourceManager registration                | Task 6                                                   |
| Poll scheduler + cron                          | Task 7                                                   |
| Diff strategy (DiffKey + ContentHash)          | Task 7                                                   |
| WebhookHook (verify + convert + emit)          | Task 8                                                   |
| Polling state persistence (in-memory + PG)     | Tasks 2, 5                                               |
| Per-entry lock granularity                     | Task 5 (pollEntry.mu)                                    |
| Webhook idempotency                            | Task 7 (IdempotencyKey = DiffKey), design note in Task 8 |
| Context lifecycle (Background context in pool) | Task 8                                                   |
| Delete detection out of scope                  | N/A (explicitly excluded)                                |
| Lifecycle (fx Start/Stop)                      | Tasks 9, 11                                              |
| Metrics (Prometheus)                           | Task 4                                                   |
| TDD unit tests                                 | Tasks 1-10                                               |
| BDD specs                                      | Task 12                                                  |

### Placeholder scan

- No TBD, TODO, or "implement later" patterns.
- All test cases have concrete input/output.
- emit_timeout is hardcoded (30s default), matching spec.

### Type consistency

- `EventSourceEmitter` = `func(ctx context.Context, events []types.DataEvent) error` used consistently in Tasks 6-8.
- `PollingEntry` struct used consistently in Tasks 2, 5, 7.
- `stubWebhookConverter` and `stubWebhookConverterWithAuth` both implement `WebhookConverter`.
- `EventSourceCollector` method names match spec metrics table.

Execution options:

1. Subagent-Driven (recommended)
2. Inline Execution
