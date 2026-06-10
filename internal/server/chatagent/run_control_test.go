package chatagent

import (
	"context"
	"testing"

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
