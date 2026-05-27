package ability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/types"
)

// --- Shared stubs ---

type stubWebhookConverter struct {
	path string
}

func (s *stubWebhookConverter) WebhookPath() string { return s.path }
func (*stubWebhookConverter) VerifySignature(_ map[string]string, _ []byte) error {
	return nil
}
func (*stubWebhookConverter) Convert(_ []byte, _ map[string]string) ([]types.DataEvent, error) {
	return nil, nil
}

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
func (*stubWebhookConverterWithAuth) Convert(_ []byte, _ map[string]string) ([]types.DataEvent, error) {
	return nil, nil
}

type countingEmitter struct {
	mu     sync.Mutex
	events []types.DataEvent
}

func (e *countingEmitter) Emit(_ context.Context, events []types.DataEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, events...)
	return nil
}

type testResource struct {
	diffKeyFn     func(any) string
	contentHashFn func(any) string
}

func (*testResource) ResourceName() string           { return "test/rsrc" }
func (*testResource) DefaultInterval() time.Duration { return time.Minute }
func (r *testResource) DiffKey(item any) string      { return r.diffKeyFn(item) }
func (r *testResource) ContentHash(item any) string  { return r.contentHashFn(item) }
func (*testResource) CursorField() string            { return "id" }
func (*testResource) List(_ context.Context, _ string) (PollResult, error) {
	return PollResult{}, nil
}

type stubPollingResource struct {
	name     string
	interval time.Duration
	items    []any
	cursor   string
}

func (s *stubPollingResource) ResourceName() string           { return s.name }
func (s *stubPollingResource) DefaultInterval() time.Duration { return s.interval }
func (*stubPollingResource) DiffKey(item any) string {
	if v, ok := item.(string); ok {
		return v
	}
	return ""
}
func (*stubPollingResource) ContentHash(item any) string {
	if v, ok := item.(string); ok {
		return v
	}
	return ""
}
func (*stubPollingResource) CursorField() string { return "id" }
func (s *stubPollingResource) List(_ context.Context, _ string) (PollResult, error) {
	return PollResult{Items: s.items, NextCursor: s.cursor}, nil
}

type mockPersistence struct {
	data       map[string]PollingEntry
	saveCalls  []saveCall
	loadAllErr error
}

type saveCall struct {
	resourceName string
	cursor       string
	knownHashes  map[string]string
}

func (m *mockPersistence) LoadAll(_ context.Context) (map[string]PollingEntry, error) {
	if m.loadAllErr != nil {
		return nil, m.loadAllErr
	}
	return m.data, nil
}

func (m *mockPersistence) Save(_ context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	if m.data == nil {
		m.data = make(map[string]PollingEntry)
	}
	m.data[resourceName] = PollingEntry{
		Cursor:      cursor,
		KnownHashes: knownHashes,
	}
	m.saveCalls = append(m.saveCalls, saveCall{
		resourceName: resourceName,
		cursor:       cursor,
		knownHashes:  knownHashes,
	})
	return nil
}

// --- PollResult tests ---

func TestPollResult_Fields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		pr          PollResult
		wantLen     int
		wantCursor  string
		wantHasMore bool
	}{
		{
			name: "with items and cursor",
			pr: PollResult{
				Items:      []any{"a", "b"},
				NextCursor: "cursor-next",
				HasMore:    true,
			},
			wantLen:     2,
			wantCursor:  "cursor-next",
			wantHasMore: true,
		},
		{
			name: "empty result",
			pr: PollResult{
				Items:      nil,
				NextCursor: "",
				HasMore:    false,
			},
			wantLen:     0,
			wantCursor:  "",
			wantHasMore: false,
		},
		{
			name: "single item no more",
			pr: PollResult{
				Items:      []any{42},
				NextCursor: "c42",
				HasMore:    false,
			},
			wantLen:     1,
			wantCursor:  "c42",
			wantHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := len(tt.pr.Items); got != tt.wantLen {
				t.Errorf("len(Items) = %d, want %d", got, tt.wantLen)
			}
			if got := tt.pr.NextCursor; got != tt.wantCursor {
				t.Errorf("NextCursor = %q, want %q", got, tt.wantCursor)
			}
			if got := tt.pr.HasMore; got != tt.wantHasMore {
				t.Errorf("HasMore = %v, want %v", got, tt.wantHasMore)
			}
		})
	}
}

func TestWebhookConverter_Interface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "has WebhookPath method"},
		{name: "has VerifySignature method"},
		{name: "has Convert method"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var _ WebhookConverter = nil // compile-time check
		})
	}
}

func TestPollingResource_Interface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "has ResourceName method"},
		{name: "has DefaultInterval method"},
		{name: "has DiffKey method"},
		{name: "has ContentHash method"},
		{name: "has CursorField method"},
		{name: "has List method"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var _ PollingResource = nil // compile-time check
		})
	}
}

// --- EventSourceManager tests ---

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
				mgr.RegisterPolling(&stubPollingResource{name: name, interval: 1 * time.Minute})
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
	})

	err := mgr.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	err = mgr.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// --- Diff tests ---

func TestDiffNewItems(t *testing.T) {
	tests := []struct {
		name       string
		known      map[string]string
		items      []any
		diffKeyFn  func(any) string
		hashFn     func(any) string
		wantEvents int
		wantTypes  []string
	}{
		{
			name:  "all new items emit created",
			known: map[string]string{},
			items: []any{"a", "b", "c"},
			diffKeyFn: func(item any) string {
				s, ok := item.(string)
				if ok {
					return s
				}
				return ""
			},
			hashFn: func(item any) string {
				s, ok := item.(string)
				if ok {
					return "h_" + s
				}
				return ""
			},
			wantEvents: 3,
			wantTypes:  []string{"test/rsrc.created", "test/rsrc.created", "test/rsrc.created"},
		},
		{
			name:  "all known items skip",
			known: map[string]string{"a": "h_a", "b": "h_b"},
			items: []any{"a", "b"},
			diffKeyFn: func(item any) string {
				s, ok := item.(string)
				if ok {
					return s
				}
				return ""
			},
			hashFn: func(item any) string {
				s, ok := item.(string)
				if ok {
					return "h_" + s
				}
				return ""
			},
			wantEvents: 0,
			wantTypes:  nil,
		},
		{
			name:  "changed item emits updated",
			known: map[string]string{"a": "old_hash", "b": "h_b"},
			items: []any{"a", "b"},
			diffKeyFn: func(item any) string {
				s, ok := item.(string)
				if ok {
					return s
				}
				return ""
			},
			hashFn: func(item any) string {
				s, ok := item.(string)
				if ok {
					return "h_" + s
				}
				return ""
			},
			wantEvents: 1,
			wantTypes:  []string{"test/rsrc.updated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &testResource{diffKeyFn: tt.diffKeyFn, contentHashFn: tt.hashFn}
			emitter := &countingEmitter{}
			entry := &pollEntry{
				resource:    r,
				knownHashes: copyMap(tt.known),
			}
			mgr := &EventSourceManager{
				emitter: emitter.Emit,
			}
			mgr.diffAndEmit(context.Background(), entry, tt.items)
			if len(emitter.events) != tt.wantEvents {
				t.Errorf("events emitted = %d, want %d", len(emitter.events), tt.wantEvents)
			}
			for i, ev := range emitter.events {
				if i < len(tt.wantTypes) && ev.EventType != tt.wantTypes[i] {
					t.Errorf("event[%d] EventType = %q, want %q", i, ev.EventType, tt.wantTypes[i])
				}
			}
		})
	}
}

// --- PollingState tests ---

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
	tests := []struct {
		name    string
		setup   func(*PollingState, *mockPersistence)
		want    map[string]PollingEntry
		wantLen int
	}{
		{
			name: "flush single resource",
			setup: func(state *PollingState, _ *mockPersistence) {
				state.Update("test/rsrc", PollingEntry{
					Cursor:      "cursor-1",
					KnownHashes: map[string]string{"k1": "h1"},
				})
			},
			want: map[string]PollingEntry{
				"test/rsrc": {Cursor: "cursor-1", KnownHashes: map[string]string{"k1": "h1"}},
			},
			wantLen: 1,
		},
		{
			name: "flush multiple resources",
			setup: func(state *PollingState, _ *mockPersistence) {
				state.Update("a/rsrc", PollingEntry{
					Cursor:      "cur-a",
					KnownHashes: map[string]string{"ka": "ha"},
				})
				state.Update("b/rsrc", PollingEntry{
					Cursor:      "cur-b",
					KnownHashes: map[string]string{"kb": "hb"},
				})
			},
			want: map[string]PollingEntry{
				"a/rsrc": {Cursor: "cur-a", KnownHashes: map[string]string{"ka": "ha"}},
				"b/rsrc": {Cursor: "cur-b", KnownHashes: map[string]string{"kb": "hb"}},
			},
			wantLen: 2,
		},
		{
			name: "flush empty dirty set is no-op",
			setup: func(_ *PollingState, _ *mockPersistence) {
			},
			want:    map[string]PollingEntry{},
			wantLen: 0,
		},
		{
			name: "flush with updated entry",
			setup: func(state *PollingState, _ *mockPersistence) {
				state.Update("test/rsrc", PollingEntry{
					Cursor:      "cursor-old",
					KnownHashes: map[string]string{"k1": "h1"},
				})
				state.Update("test/rsrc", PollingEntry{
					Cursor:      "cursor-new",
					KnownHashes: map[string]string{"k2": "h2"},
				})
			},
			want: map[string]PollingEntry{
				"test/rsrc": {Cursor: "cursor-new", KnownHashes: map[string]string{"k2": "h2"}},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persist := &mockPersistence{}
			state := NewPollingState(persist)
			tt.setup(state, persist)

			err := state.Flush(context.Background())
			if err != nil {
				t.Fatalf("Flush: %v", err)
			}

			if len(persist.data) != tt.wantLen {
				t.Errorf("persisted data len = %d, want %d", len(persist.data), tt.wantLen)
			}
			for key, wantEntry := range tt.want {
				got, ok := persist.data[key]
				if !ok {
					t.Errorf("missing key %q in persisted data", key)
					continue
				}
				if got.Cursor != wantEntry.Cursor {
					t.Errorf("cursor for %q = %q, want %q", key, got.Cursor, wantEntry.Cursor)
				}
			}
		})
	}
}

func TestPollingState_Load(t *testing.T) {
	tests := []struct {
		name      string
		seed      map[string]PollingEntry
		preEntry  string
		preCursor string
	}{
		{
			name:      "load from empty backend",
			seed:      nil,
			preEntry:  "pre/loaded",
			preCursor: "old-cursor",
		},
		{
			name: "load from backend with data",
			seed: map[string]PollingEntry{
				"a/rsrc": {Cursor: "cur-a", KnownHashes: map[string]string{"ka": "ha"}},
			},
			preEntry:  "b/rsrc",
			preCursor: "old-cursor",
		},
		{
			name: "load overwrites existing entries",
			seed: map[string]PollingEntry{
				"x/rsrc": {Cursor: "persisted-cursor", KnownHashes: map[string]string{"kx": "hx"}},
			},
			preEntry:  "x/rsrc",
			preCursor: "stale-cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persist := &mockPersistence{data: tt.seed}
			state := NewPollingState(persist)

			state.Update(tt.preEntry, PollingEntry{Cursor: tt.preCursor})

			err := state.Load(context.Background())
			if err != nil {
				t.Fatalf("Load: %v", err)
			}

			preEntry := state.Get(tt.preEntry)
			if tt.preEntry == "x/rsrc" && tt.seed["x/rsrc"].Cursor == "persisted-cursor" {
				if preEntry.Cursor != "persisted-cursor" {
					t.Errorf("entry %q cursor after load = %q, want %q (should be overwritten by persisted)", tt.preEntry, preEntry.Cursor, "persisted-cursor")
				}
			} else {
				if preEntry.Cursor != tt.preCursor {
					t.Errorf("entry %q cursor after load = %q, want %q", tt.preEntry, preEntry.Cursor, tt.preCursor)
				}
			}

			for key, wantEntry := range tt.seed {
				got := state.Get(key)
				if got.Cursor != wantEntry.Cursor {
					t.Errorf("loaded entry %q cursor = %q, want %q", key, got.Cursor, wantEntry.Cursor)
				}
			}
		})
	}
}

func TestPollingState_MarkDirty(t *testing.T) {
	tests := []struct {
		name  string
		marks []string
		want  []string
	}{
		{
			name:  "mark single resource dirty",
			marks: []string{"a/rsrc"},
			want:  []string{"a/rsrc"},
		},
		{
			name:  "mark multiple resources dirty",
			marks: []string{"a/rsrc", "b/rsrc", "c/rsrc"},
			want:  []string{"a/rsrc", "b/rsrc", "c/rsrc"},
		},
		{
			name:  "mark already dirty resource",
			marks: []string{"a/rsrc", "a/rsrc", "a/rsrc"},
			want:  []string{"a/rsrc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persist := &mockPersistence{}
			state := NewPollingState(persist)

			for _, name := range tt.marks {
				state.Update(name, PollingEntry{Cursor: "c"})
				state.MarkDirty(name)
			}

			err := state.Flush(context.Background())
			if err != nil {
				t.Fatalf("Flush: %v", err)
			}

			if len(persist.data) != len(tt.want) {
				t.Errorf("persisted data len = %d, want %d", len(persist.data), len(tt.want))
			}
			for _, name := range tt.want {
				if _, ok := persist.data[name]; !ok {
					t.Errorf("missing key %q in persisted data", name)
				}
			}
		})
	}
}

// --- Webhook handler tests ---

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
		verifyFn: func(_ map[string]string, _ []byte) error {
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

func TestWebhookHandler_Success(t *testing.T) {
	app := fiber.New()
	mgr := NewEventSourceManager(nil, nil, nil)
	mgr.RegisterWebhook(&stubWebhookConverterWithAuth{
		path:     "github/events",
		verifyFn: nil, // nil = no error = pass
	})
	app.Post("/webhook/provider/*", mgr.WebhookHandler())

	tests := []struct {
		name string
		body string
	}{
		{name: "valid request returns 202", body: `{"action": "created"}`},
		{name: "empty body returns 202", body: ``},
		{name: "large payload returns 202", body: `{"data": "` + strings.Repeat("x", 5000) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook/provider/github/events",
				strings.NewReader(tt.body))
			resp, _ := app.Test(req)
			if resp.StatusCode != fiber.StatusAccepted {
				t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusAccepted)
			}
		})
	}
}

// TestWebhookHandler_LowercaseHeadersCanonicalized verifies that the
// WebhookHandler canonicalizes HTTP header keys before passing them to
// VerifySignature. This prevents 401 failures when reverse proxies normalize
// header casing (Bug 1 - High).
func TestWebhookHandler_LowercaseHeadersCanonicalized(t *testing.T) {
	app := fiber.New()
	mgr := NewEventSourceManager(nil, nil, nil)

	var capturedHeaders map[string]string
	mgr.RegisterWebhook(&stubWebhookConverterWithAuth{
		path: "test/events",
		verifyFn: func(headers map[string]string, _ []byte) error {
			capturedHeaders = headers
			if _, ok := headers["X-Test-Signature"]; !ok {
				return errors.New("missing X-Test-Signature header")
			}
			return nil
		},
	})
	app.Post("/webhook/provider/*", mgr.WebhookHandler())

	// Submit a request where HTTP headers use mixed/lowercase casing to
	// simulate the effect of reverse proxies that normalize casing.
	// Go's net/http canonicalizes keys on Set/Add, so the raw map is
	// assigned directly to preserve the lowercase originals.
	req := httptest.NewRequest("POST", "/webhook/provider/test/events", strings.NewReader("body"))
	req.Header = http.Header{
		"x-test-signature": {"test-value"},
		"x-another-header": {"other-value"},
		"Content-Type":     {"application/json"},
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusAccepted {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusAccepted)
	}
	if capturedHeaders == nil {
		t.Fatal("capturedHeaders is nil, verifyFn was never called")
	}
	if got := capturedHeaders["X-Test-Signature"]; got != "test-value" {
		t.Errorf("X-Test-Signature = %q, want %q (lowercase x-test-signature should be canonicalized)", got, "test-value")
	}
	if got := capturedHeaders["X-Another-Header"]; got != "other-value" {
		t.Errorf("X-Another-Header = %q, want %q (lowercase x-another-header should be canonicalized)", got, "other-value")
	}
	if got := capturedHeaders["Content-Type"]; got != "application/json" {
		t.Errorf("Content-Type = %q, want %q (canonical key should not be altered)", got, "application/json")
	}
}
