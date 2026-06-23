package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelRun(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cancels bound context"},
		{name: "no-op when unbound"},
		{name: "rebind cancels previous"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "cancels bound context":
				ctx, cancel := context.WithCancel(context.Background())
				registerRunCancel("sess-a", cancel)
				t.Cleanup(func() { unregisterRunCancel("sess-a") })
				cancelRun("sess-a")
				require.ErrorIs(t, ctx.Err(), context.Canceled)
			case "no-op when unbound":
				assert.NotPanics(t, func() { cancelRun("missing-session") })
			case "rebind cancels previous":
				ctx1, cancel1 := context.WithCancel(context.Background())
				ctx2, cancel2 := context.WithCancel(context.Background())
				registerRunCancel("sess-b", cancel1)
				registerRunCancel("sess-b", cancel2)
				t.Cleanup(func() { unregisterRunCancel("sess-b") })
				require.ErrorIs(t, ctx1.Err(), context.Canceled)
				require.NoError(t, ctx2.Err())
			}
		})
	}
}

func TestReleaseSessionLock(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "lock can be re-created after release"},
		{name: "release unknown session is safe"},
		{name: "same session shares lock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "lock can be re-created after release":
				releaseSessionLock("sess-lock-1")
				first := sessionLock("sess-lock-1")
				releaseSessionLock("sess-lock-1")
				second := sessionLock("sess-lock-1")
				assert.NotSame(t, first, second)
			case "release unknown session is safe":
				assert.NotPanics(t, func() { releaseSessionLock("unknown-lock") })
			case "same session shares lock":
				a := sessionLock("sess-lock-2")
				b := sessionLock("sess-lock-2")
				assert.Same(t, a, b)
				releaseSessionLock("sess-lock-2")
			}
		})
	}
}

func TestEvictStaleRunCancel(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "stale run cancel entry is removed"},
		{name: "fresh run cancel entry is kept"},
		{name: "evict unknown session is safe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "stale run cancel entry is removed":
				ctx, cancel := context.WithCancel(context.Background())
				runCancelsMu.Lock()
				runCancels["stale-cancel"] = &runCancelEntry{
					cancel:   cancel,
					lastUsed: time.Now().Add(-sessionLockTTL - time.Minute),
				}
				runCancelsMu.Unlock()
				t.Cleanup(func() {
					unregisterRunCancel("stale-cancel")
					cancel()
				})

				runCancelsMu.Lock()
				now := time.Now()
				for id, entry := range runCancels {
					if now.Sub(entry.lastUsed) > sessionLockTTL {
						delete(runCancels, id)
					}
				}
				_, ok := runCancels["stale-cancel"]
				runCancelsMu.Unlock()
				assert.False(t, ok)
				require.NoError(t, ctx.Err())
			case "fresh run cancel entry is kept":
				ctx, cancel := context.WithCancel(context.Background())
				registerRunCancel("fresh-cancel", cancel)
				t.Cleanup(func() { unregisterRunCancel("fresh-cancel") })

				runCancelsMu.Lock()
				_, ok := runCancels["fresh-cancel"]
				runCancelsMu.Unlock()
				assert.True(t, ok)
				require.NoError(t, ctx.Err())
			case "evict unknown session is safe":
				assert.NotPanics(t, func() { unregisterRunCancel("never-bound") })
			}
		})
	}
}
