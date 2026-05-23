package ability

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

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
func (*testResource) DefaultInterval() time.Duration  { return time.Minute }
func (r *testResource) DiffKey(item any) string         { return r.diffKeyFn(item) }
func (r *testResource) ContentHash(item any) string     { return r.contentHashFn(item) }
func (*testResource) CursorField() string             { return "id" }
func (*testResource) List(_ context.Context, _ string) (PollResult, error) {
	return PollResult{}, nil
}

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
		name:       "all new items emit created",
		known:      map[string]string{},
		items:      []any{"a", "b", "c"},
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
		name:       "all known items skip",
		known:      map[string]string{"a": "h_a", "b": "h_b"},
		items:      []any{"a", "b"},
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
		name:       "changed item emits updated",
		known:      map[string]string{"a": "old_hash", "b": "h_b"},
		items:      []any{"a", "b"},
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
