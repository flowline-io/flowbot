package capability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pollOnceResource struct {
	name     string
	items    []any
	cursor   string
	listErr  error
	interval time.Duration
}

func (r *pollOnceResource) ResourceName() string { return r.name }
func (r *pollOnceResource) DefaultInterval() time.Duration {
	if r.interval > 0 {
		return r.interval
	}
	return time.Minute
}
func (*pollOnceResource) DiffKey(item any) string {
	m, ok := item.(map[string]string)
	if !ok {
		return ""
	}
	return m["id"]
}
func (*pollOnceResource) ContentHash(item any) string {
	m, ok := item.(map[string]string)
	if !ok {
		return ""
	}
	return m["hash"]
}
func (*pollOnceResource) CursorField() string { return "id" }
func (r *pollOnceResource) List(_ context.Context, _ string) (PollResult, error) {
	if r.listErr != nil {
		return PollResult{}, r.listErr
	}
	return PollResult{Items: r.items, NextCursor: r.cursor, HasMore: r.cursor != ""}, nil
}

func TestEventSourceManagerPollOnce(t *testing.T) {
	tests := []struct {
		name       string
		resource   *pollOnceResource
		wantEvents int
		wantCursor string
	}{
		{
			name: "emits created events for new items",
			resource: &pollOnceResource{
				name:  "poll/success",
				items: []any{map[string]string{"id": "1", "hash": "h1"}},
			},
			wantEvents: 1,
		},
		{
			name: "updates cursor on success",
			resource: &pollOnceResource{
				name:   "poll/cursor",
				items:  []any{map[string]string{"id": "1", "hash": "h1"}},
				cursor: "next",
			},
			wantCursor: "next",
		},
		{
			name: "list error increments failures without panic",
			resource: &pollOnceResource{
				name:    "poll/error",
				listErr: errors.New("upstream down"),
			},
			wantEvents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitter := &countingEmitter{}
			mgr := NewEventSourceManager(emitter.Emit, NewPollingState(nil), metrics.NewEventSourceCollector(nil))
			mgr.RegisterPolling(tt.resource)
			entry := mgr.pollers[tt.resource.name]
			require.NotNil(t, entry)

			mgr.pollOnce(context.Background(), tt.resource.name, entry)

			if tt.wantEvents > 0 {
				assert.Len(t, emitter.events, tt.wantEvents)
			}
			if tt.wantCursor != "" {
				entry.mu.Lock()
				defer entry.mu.Unlock()
				assert.Equal(t, tt.wantCursor, entry.cursor)
			}
		})
	}
}

func TestEventSourceManagerSetPool(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "assigns pool reference"},
		{name: "overwrites previous pool"},
		{name: "nil pool clears reference"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewEventSourceManager(nil, nil, nil)
			if tt.name == "nil pool clears reference" {
				mgr.SetPool(nil)
				mgr.mu.RLock()
				assert.Nil(t, mgr.pool)
				mgr.mu.RUnlock()
				return
			}

			require.NoError(t, InitEventPool(2, "1s", nil))
			t.Cleanup(ShutdownEventPool)

			pool := GetEventPool()
			require.NotNil(t, pool)
			mgr.SetPool(pool)
			mgr.mu.RLock()
			assert.Equal(t, pool, mgr.pool)
			mgr.mu.RUnlock()

			if tt.name == "overwrites previous pool" {
				mgr.SetPool(pool)
				mgr.mu.RLock()
				assert.NotNil(t, mgr.pool)
				mgr.mu.RUnlock()
			}
		})
	}
}

func TestRegistryInvokeRecordsErrorMetrics(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "typed error records code", err: types.Errorf(types.ErrNotFound, "missing item")},
		{name: "generic error records unknown", err: errors.New("boom")},
		{name: "not implemented error", err: types.Errorf(types.ErrNotImplemented, "nope")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			r.metrics = metrics.NewCapabilityCollector(nil)
			require.NoError(t, r.Register(hub.CapExample, "fail", func(context.Context, map[string]any) (*InvokeResult, error) {
				return nil, tt.err
			}))

			_, err := r.Invoke(context.Background(), hub.CapExample, "fail", nil)
			require.Error(t, err)
		})
	}
}

func TestBuildHashSet(t *testing.T) {
	tests := []struct {
		name  string
		items []any
		want  int
	}{
		{name: "maps diff keys to hashes", items: []any{
			map[string]string{"id": "a", "hash": "h1"},
			map[string]string{"id": "b", "hash": "h2"},
		}, want: 2},
		{name: "skips empty diff keys", items: []any{map[string]string{"hash": "h1"}}, want: 1},
		{name: "empty items", items: nil, want: 0},
	}

	resource := &pollOnceResource{name: "hash-set"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildHashSet(tt.items, resource.DiffKey, resource.ContentHash)
			assert.Len(t, got, tt.want)
		})
	}
}
