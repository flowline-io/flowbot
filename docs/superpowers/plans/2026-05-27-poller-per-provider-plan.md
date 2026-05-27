# Poller Per-Provider Relocation - Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move `PollingResource` implementations from capability-level packages into per-provider adapter directories, and consolidate poller registration in `hub/module.go` `Bootstrap()` alongside webhook registrations.

**Architecture:** Poller files (`poller.go`) are relocated to adapter directories alongside `adapter.go` and `webhook.go`. Constructors renamed to `NewPoller()` (matching `NewWebhook()` convention) and create their own service internally. Registration moves from `pipeline.go` (hardcoded adapter imports) to `hub/module.go` `Bootstrap()`.

**Tech Stack:** Go 1.26+, testify

---

### Task 1: Relocate ExamplePoller to adapter directory

**Files:**

- Create: `pkg/ability/example/example/poller.go`
- Delete: `pkg/ability/example/poller.go`

- [ ] **Step 1: Create poller.go in adapter directory**

Write `pkg/ability/example/example/poller.go`:

```go
// Package example implements the example provider adapter for the example capability.
package example

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	exsvc "github.com/flowline-io/flowbot/pkg/ability/example"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ExamplePoller implements ability.PollingResource for the example provider.
// It polls the example provider for new and updated items via the example Service.
type ExamplePoller struct {
	svc     exsvc.Service
	secret  []byte
	nowFunc func() time.Time
}

// NewPoller creates an ExamplePoller backed by a default adapter.
func NewPoller() ability.PollingResource {
	return &ExamplePoller{
		svc:     New(),
		secret:  []byte("example-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// NewPollerWithService creates an ExamplePoller with a specific service, useful for testing.
func NewPollerWithService(svc exsvc.Service) *ExamplePoller {
	return &ExamplePoller{
		svc:     svc,
		secret:  []byte("example-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// ResourceName returns the unique name for this polling resource.
func (*ExamplePoller) ResourceName() string {
	return "example/events"
}

// DefaultInterval returns the recommended polling interval.
func (*ExamplePoller) DefaultInterval() time.Duration {
	return 60 * time.Second
}

// DiffKey returns the unique identifier for an item, used for change detection.
func (*ExamplePoller) DiffKey(item any) string {
	if m, ok := item.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}
	return fmt.Sprintf("%v", item)
}

// ContentHash returns a SHA256 hash of the item for content-based change detection.
func (*ExamplePoller) ContentHash(item any) string {
	data := fmt.Sprintf("%v", item)
	h := sha256.New()
	_, _ = h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CursorField returns the field name used for cursor-based pagination.
func (*ExamplePoller) CursorField() string {
	return "cursor"
}

// List fetches a batch of items from the provider starting after the given cursor.
func (p *ExamplePoller) List(ctx context.Context, cursor string) (ability.PollResult, error) {
	if err := ctx.Err(); err != nil {
		return ability.PollResult{}, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, nextCursor, err := p.svc.ListRawEvents(ctx, cursor)
	if err != nil {
		return ability.PollResult{}, err
	}
	return ability.PollResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}, nil
}

// Compile-time check that ExamplePoller implements ability.PollingResource.
var _ ability.PollingResource = (*ExamplePoller)(nil)
```

- [ ] **Step 2: Run tests to verify compilation**

```bash
go build ./pkg/ability/example/example/...
```

Expected: no compile errors.

- [ ] **Step 3: Delete old poller file**

```bash
git rm pkg/ability/example/poller.go
```

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "refactor: relocate ExamplePoller to adapter directory"
```

---

### Task 2: Relocate ExamplePoller tests to adapter directory

**Files:**

- Create: `pkg/ability/example/example/poller_test.go`
- Delete: `pkg/ability/example/poller_test.go`

- [ ] **Step 1: Create poller_test.go in adapter directory**

Write `pkg/ability/example/example/poller_test.go`:

```go
package example

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	exsvc "github.com/flowline-io/flowbot/pkg/ability/example"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

type fakePollerService struct {
	items  []any
	cursor string
	err    error
}

func (*fakePollerService) GetItem(_ context.Context, _ string) (*ability.Host, error) {
	return nil, nil
}
func (*fakePollerService) ListItems(_ context.Context, _ *exsvc.ListQuery) (*ability.ListResult[ability.Host], error) {
	return nil, nil
}
func (*fakePollerService) CreateItem(_ context.Context, _ string, _ types.KV) (*ability.Host, error) {
	return nil, nil
}
func (*fakePollerService) UpdateItem(_ context.Context, _ string, _ map[string]any) (*ability.Host, error) {
	return nil, nil
}
func (*fakePollerService) DeleteItem(_ context.Context, _ string) error { return nil }
func (*fakePollerService) HealthCheck(_ context.Context) (bool, error)  { return true, nil }
func (f *fakePollerService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return f.items, f.cursor, f.err
}

func TestExamplePoller_ResourceName(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakePollerService{})
	assert.Equal(t, "example/events", p.ResourceName())
}

func TestExamplePoller_DefaultInterval(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakePollerService{})
	assert.Equal(t, 60*time.Second, p.DefaultInterval())
}

func TestExamplePoller_CursorField(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakePollerService{})
	assert.Equal(t, "cursor", p.CursorField())
}

func TestExamplePoller_DiffKey(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakePollerService{})
	tests := []struct {
		name string
		item any
		want string
	}{
		{name: "map with id field", item: map[string]any{"id": "abc-123"}, want: "abc-123"},
		{name: "map without id field", item: map[string]any{"key": "val"}, want: "map[key:val]"},
		{name: "string item", item: "plain-string", want: "plain-string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := p.DiffKey(tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExamplePoller_ContentHash(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakePollerService{})
	tests := []struct {
		name string
		a    any
		b    any
		same bool
	}{
		{name: "same items produce same hash", a: map[string]any{"id": "1"}, b: map[string]any{"id": "1"}, same: true},
		{name: "different items produce different hash", a: map[string]any{"id": "1"}, b: map[string]any{"id": "2"}, same: false},
		{name: "hash is non-empty", a: map[string]any{"id": "x"}, same: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash := p.ContentHash(tt.a)
			assert.NotEmpty(t, hash)
			if tt.same && tt.name == "same items produce same hash" {
				assert.Equal(t, hash, p.ContentHash(tt.b))
			}
			if !tt.same && tt.name == "different items produce different hash" {
				assert.NotEqual(t, hash, p.ContentHash(tt.b))
			}
		})
	}
}

func TestExamplePoller_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		svc        *fakePollerService
		cursor     string
		wantItems  int
		wantCursor string
		wantMore   bool
		wantErr    bool
	}{
		{
			name:       "returns items with no cursor",
			svc:        &fakePollerService{items: []any{map[string]any{"id": "1"}}},
			wantItems:  1,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
		{
			name:       "returns items with next cursor",
			svc:        &fakePollerService{items: []any{map[string]any{"id": "1"}}, cursor: "next-page"},
			wantItems:  1,
			wantCursor: "next-page",
			wantMore:   true,
			wantErr:    false,
		},
		{
			name:       "empty result",
			svc:        &fakePollerService{items: []any{}},
			wantItems:  0,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewPollerWithService(tt.svc)
			result, err := p.List(context.Background(), tt.cursor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Items, tt.wantItems)
			assert.Equal(t, tt.wantCursor, result.NextCursor)
			assert.Equal(t, tt.wantMore, result.HasMore)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
go test ./pkg/ability/example/example/ -run "TestExamplePoller" -v
```

Expected: all tests pass.

- [ ] **Step 3: Delete old test file**

```bash
git rm pkg/ability/example/poller_test.go
```

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "refactor: relocate ExamplePoller tests to adapter directory"
```

---

### Task 3: Relocate NotePoller to adapter directory

**Files:**

- Create: `pkg/ability/note/trilium/poller.go`
- Delete: `pkg/ability/note/poller.go`

- [ ] **Step 1: Create poller.go in adapter directory**

Write `pkg/ability/note/trilium/poller.go`:

```go
// Package trilium implements the Trilium adapter for the note capability.
package trilium

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	notesvc "github.com/flowline-io/flowbot/pkg/ability/note"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

// NotePoller implements ability.PollingResource for the note capability.
// It polls Trilium for new and updated notes.
type NotePoller struct {
	svc     notesvc.Service
	secret  []byte
	nowFunc func() time.Time
}

// NewPoller creates a NotePoller backed by a default adapter.
func NewPoller() ability.PollingResource {
	return &NotePoller{
		svc:     New(),
		secret:  []byte("note-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// NewPollerWithService creates a NotePoller with a specific service, useful for testing.
func NewPollerWithService(svc notesvc.Service) *NotePoller {
	return &NotePoller{
		svc:     svc,
		secret:  []byte("note-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// ResourceName returns the unique name for this polling resource.
func (*NotePoller) ResourceName() string {
	return "note/events"
}

// DefaultInterval returns the recommended polling interval.
func (*NotePoller) DefaultInterval() time.Duration {
	return 120 * time.Second
}

// DiffKey returns the unique identifier for an item, used for change detection.
func (*NotePoller) DiffKey(item any) string {
	if m, ok := item.(map[string]any); ok {
		if id, ok := m["noteId"].(string); ok {
			return id
		}
	}
	return fmt.Sprintf("%v", item)
}

// ContentHash returns a SHA256 hash of the item for content-based change detection.
func (*NotePoller) ContentHash(item any) string {
	data := fmt.Sprintf("%v", item)
	h := sha256.New()
	_, _ = h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CursorField returns the field name used for cursor-based pagination.
func (*NotePoller) CursorField() string {
	return "cursor"
}

// List fetches a batch of items from the provider starting after the given cursor.
func (p *NotePoller) List(ctx context.Context, cursor string) (ability.PollResult, error) {
	if err := ctx.Err(); err != nil {
		return ability.PollResult{}, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, nextCursor, err := p.svc.ListRawEvents(ctx, cursor)
	if err != nil {
		return ability.PollResult{}, err
	}
	return ability.PollResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}, nil
}

// Compile-time check that NotePoller implements ability.PollingResource.
var _ ability.PollingResource = (*NotePoller)(nil)
```

- [ ] **Step 2: Run build to verify compilation**

```bash
go build ./pkg/ability/note/trilium/...
```

Expected: no compile errors.

- [ ] **Step 3: Delete old poller file**

```bash
git rm pkg/ability/note/poller.go
```

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "refactor: relocate NotePoller to trilium adapter directory"
```

---

### Task 4: Relocate NotePoller tests to adapter directory

**Files:**

- Create: `pkg/ability/note/trilium/poller_test.go`
- Delete: `pkg/ability/note/poller_test.go`

- [ ] **Step 1: Create poller_test.go in adapter directory**

Write `pkg/ability/note/trilium/poller_test.go`:

```go
package trilium

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notesvc "github.com/flowline-io/flowbot/pkg/ability/note"
	"github.com/flowline-io/flowbot/pkg/ability"
)

type fakeNotePollerService struct {
	items  []any
	cursor string
	err    error
}

func (*fakeNotePollerService) List(_ context.Context, _ *notesvc.ListQuery) (*ability.ListResult[ability.Note], error) {
	return nil, nil
}
func (*fakeNotePollerService) Get(_ context.Context, _ string) (*ability.Note, error) { return nil, nil }
func (*fakeNotePollerService) Create(_ context.Context, _, _, _, _ string) (*ability.Note, error) {
	return nil, nil
}
func (*fakeNotePollerService) Update(_ context.Context, _, _, _ string) (*ability.Note, error) {
	return nil, nil
}
func (*fakeNotePollerService) Delete(_ context.Context, _ string) error            { return nil }
func (*fakeNotePollerService) GetContent(_ context.Context, _ string) (string, error) { return "", nil }
func (*fakeNotePollerService) SetContent(_ context.Context, _, _ string) error      { return nil }
func (*fakeNotePollerService) Search(_ context.Context, _ string) (*ability.ListResult[ability.Note], error) {
	return nil, nil
}
func (*fakeNotePollerService) GetAppInfo(_ context.Context) (*ability.Note, error) { return nil, nil }
func (f *fakeNotePollerService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return f.items, f.cursor, f.err
}

func TestNotePoller_ResourceName(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	assert.Equal(t, "note/events", p.ResourceName())
}

func TestNotePoller_DefaultInterval(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	assert.Equal(t, 120*time.Second, p.DefaultInterval())
}

func TestNotePoller_CursorField(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	assert.Equal(t, "cursor", p.CursorField())
}

func TestNotePoller_DiffKey(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	tests := []struct {
		name string
		item any
		want string
	}{
		{name: "map with noteId field", item: map[string]any{"noteId": "abc-123"}, want: "abc-123"},
		{name: "map without noteId field", item: map[string]any{"key": "val"}, want: "map[key:val]"},
		{name: "string item", item: "plain-string", want: "plain-string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := p.DiffKey(tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotePoller_ContentHash(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	tests := []struct {
		name string
		a    any
		b    any
		same bool
	}{
		{name: "same items produce same hash", a: map[string]any{"noteId": "1"}, b: map[string]any{"noteId": "1"}, same: true},
		{name: "different items produce different hash", a: map[string]any{"noteId": "1"}, b: map[string]any{"noteId": "2"}, same: false},
		{name: "hash is non-empty", a: map[string]any{"noteId": "x"}, same: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash := p.ContentHash(tt.a)
			assert.NotEmpty(t, hash)
			if tt.same && tt.name == "same items produce same hash" {
				assert.Equal(t, hash, p.ContentHash(tt.b))
			}
			if !tt.same && tt.name == "different items produce different hash" {
				assert.NotEqual(t, hash, p.ContentHash(tt.b))
			}
		})
	}
}

func TestNotePoller_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		svc        *fakeNotePollerService
		cursor     string
		wantItems  int
		wantCursor string
		wantMore   bool
		wantErr    bool
	}{
		{
			name:       "returns items with no cursor",
			svc:        &fakeNotePollerService{items: []any{map[string]any{"noteId": "1"}}},
			wantItems:  1,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
		{
			name:       "returns items with next cursor",
			svc:        &fakeNotePollerService{items: []any{map[string]any{"noteId": "1"}}, cursor: "next-page"},
			wantItems:  1,
			wantCursor: "next-page",
			wantMore:   true,
			wantErr:    false,
		},
		{
			name:       "empty result",
			svc:        &fakeNotePollerService{items: []any{}},
			wantItems:  0,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
		{
			name:    "service error",
			svc:     &fakeNotePollerService{err: assert.AnError},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewPollerWithService(tt.svc)
			result, err := p.List(context.Background(), tt.cursor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Items, tt.wantItems)
			assert.Equal(t, tt.wantCursor, result.NextCursor)
			assert.Equal(t, tt.wantMore, result.HasMore)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
go test ./pkg/ability/note/trilium/ -run "TestNotePoller" -v
```

Expected: all tests pass.

- [ ] **Step 3: Delete old test file**

```bash
git rm pkg/ability/note/poller_test.go
```

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "refactor: relocate NotePoller tests to trilium adapter directory"
```

---

### Task 5: Remove poller factory from example adapter

**Files:**

- Modify: `pkg/ability/example/example/adapter.go`

- [ ] **Step 1: Remove NewExamplePoller function**

Remove lines 140-142 from `pkg/ability/example/example/adapter.go`:

```go
// NewExamplePoller creates an ExamplePoller wired with a default adapter.
func NewExamplePoller() *exsvc.ExamplePoller {
	return exsvc.NewExamplePoller(New())
}
```

Also remove the unused `exsvc` import if `ListRawEvents` doesn't reference it. Verify: `ListRawEvents` (line 125-138) returns `[]any` not `exsvc.*`, so the `exsvc` import is still used by `New()`, `NewWithClient()`, `ListItems`, etc. No import change needed.

- [ ] **Step 2: Verify build**

```bash
go build ./pkg/ability/example/example/...
```

Expected: no compile errors.

- [ ] **Step 3: Remove NewNotePoller function from trilium adapter**

Remove lines 241-243 from `pkg/ability/note/trilium/adapter.go`:

```go
// NewNotePoller creates a NotePoller wired with a default adapter.
func NewNotePoller() *notesvc.NotePoller {
	return notesvc.NewNotePoller(New())
}
```

- [ ] **Step 4: Verify build**

```bash
go build ./pkg/ability/note/trilium/...
```

Expected: no compile errors.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "refactor: remove old poller factories from adapters"
```

---

### Task 6: Clean up pipeline.go

**Files:**

- Modify: `internal/server/pipeline.go`

- [ ] **Step 1: Remove example adapter import and hardcoded registrations**

Edit `internal/server/pipeline.go`:

Remove the import (line 14):

```go
exampleAdapter "github.com/flowline-io/flowbot/pkg/ability/example/example"
```

Remove the webhook registration with TODO comment (lines 212-213):

```go
srcMgr.RegisterWebhook(exampleAdapter.NewExampleWebhook()) // TODO: refactor
flog.Info("event source: registered example webhook on /webhook/provider/example")
```

Remove the poller registration (lines 215-216):

```go
srcMgr.RegisterPolling(exampleAdapter.NewExamplePoller())
flog.Info("event source: registered example poller")
```

- [ ] **Step 2: Verify build**

```bash
go build ./internal/server/...
```

Expected: no compile errors.

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "refactor: remove hardcoded webhook and poller registrations from pipeline.go"
```

---

### Task 7: Register pollers in hub module Bootstrap()

**Files:**

- Modify: `internal/modules/hub/module.go`

- [ ] **Step 1: Add adapter imports and poller registration**

Add the following imports to `internal/modules/hub/module.go`:

```go
exampleAdapter "github.com/flowline-io/flowbot/pkg/ability/example/example"
triliumAdapter "github.com/flowline-io/flowbot/pkg/ability/note/trilium"
```

Add poller registrations in `Bootstrap()` after the webhook registrations:

```go
// Pollers
mgr.RegisterPolling(exampleAdapter.NewPoller())
flog.Info("hub: registered example poller")
mgr.RegisterPolling(triliumAdapter.NewPoller())
flog.Info("hub: registered trilium note poller")
```

- [ ] **Step 2: Verify build**

```bash
go build ./internal/modules/hub/...
```

Expected: no compile errors.

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: register pollers in hub module Bootstrap alongside webhooks"
```

---

### Task 8: Update AGENTS.md

**Files:**

- Modify: `pkg/ability/AGENTS.md`

- [ ] **Step 1: Update PollingResource section**

Replace lines 215-232 (the PollingResource section) with updated content that reflects per-provider location:

Replace:

````markdown
### PollingResource (Optional)

When a provider lacks webhooks, implement `ability.PollingResource`:

```go
// pkg/ability/example/poller.go
type ExamplePoller struct { svc Service; secret []byte }

func (*ExamplePoller) ResourceName() string { ... }
func (*ExamplePoller) DefaultInterval() time.Duration { ... }
func (*ExamplePoller) DiffKey(item any) string { ... }
func (*ExamplePoller) ContentHash(item any) string { ... }
func (*ExamplePoller) CursorField() string { ... }
func (p *ExamplePoller) List(ctx context.Context, cursor string) (ability.PollResult, error) { ... }
```
````

- `Service` should expose a `ListRawEvents` method that the poller delegates to.
- Register via `ability.EventSourceManager.RegisterPollingResource()`.

````

With:
```markdown
### PollingResource (Optional)

When a provider lacks webhooks, implement `ability.PollingResource` in the adapter directory alongside `adapter.go` and `webhook.go`:

```go
// pkg/ability/<capability>/<backend>/poller.go
type NotePoller struct {
	svc     notesvc.Service
	secret  []byte
	nowFunc func() time.Time
}

// NewPoller creates a poller backed by a default adapter.
func NewPoller() ability.PollingResource {
	return &NotePoller{svc: New(), ...}
}

// NewPollerWithService creates a poller with a specific service, useful for testing.
func NewPollerWithService(svc notesvc.Service) *NotePoller {
	return &NotePoller{svc: svc, ...}
}

func (*NotePoller) ResourceName() string { ... }
func (*NotePoller) DefaultInterval() time.Duration { ... }
func (*NotePoller) DiffKey(item any) string { ... }
func (*NotePoller) ContentHash(item any) string { ... }
func (*NotePoller) CursorField() string { ... }
func (p *NotePoller) List(ctx context.Context, cursor string) (ability.PollResult, error) { ... }
````

- Include `var _ ability.PollingResource = (*NotePoller)(nil)` for compile-time safety.
- `Service` should expose a `ListRawEvents` method that the poller delegates to.
- Register via `ability.EventSourceManager.RegisterPolling()` in the hub module's `Bootstrap()` alongside webhook converters.

````

- [ ] **Step 2: Commit**

```bash
git add -A && git commit -m "docs: update AGENTS.md for per-provider poller convention"
````

---

### Task 9: Run full test and lint

- [ ] **Step 1: Run unit tests**

```bash
go tool task test
```

Expected: all tests pass.

- [ ] **Step 2: Run lint**

```bash
go tool task lint
```

Expected: no lint errors.
