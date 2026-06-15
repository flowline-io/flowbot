package chatagent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRunInterrupted(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "context canceled", err: context.Canceled, want: true},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: true},
		{name: "wrapped cancel", err: errors.Join(context.Canceled, assert.AnError), want: true},
		{name: "other error", err: assert.AnError, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isRunInterrupted(tt.err))
		})
	}
}

func TestReleaseHarnessAfterRunAbort(t *testing.T) {
	tests := []struct {
		name    string
		scripts []agentllm.ResponseScript
	}{
		{name: "single response", scripts: []agentllm.ResponseScript{{Content: "ok"}}},
		{name: "multi response", scripts: []agentllm.ResponseScript{{Content: "a"}, {Content: "b"}}},
		{name: "empty tail response", scripts: []agentllm.ResponseScript{{Content: "done"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			fakeModel := agentllm.NewFakeModel(tt.scripts...)
			h := harness.New(harness.Options{
				AgentOptions: agent.Options{Model: fakeModel},
				ModelName:    "fake",
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
			require.NoError(t, err)
			cancel()

			require.ErrorIs(t, h.WaitIdle(ctx), context.Canceled)

			releaseHarnessAfterRunAbort(h, "sess-test")
			require.NoError(t, h.WaitIdle(context.Background()))

			_, err = h.Prompt(context.Background(), agent.NewUserMessage("follow-up"))
			require.NoError(t, err)
			require.NoError(t, h.WaitIdle(context.Background()))
		})
	}
}

func TestReleaseHarnessAfterRunAbortNilHarness(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "nil harness is no-op"},
		{name: "second nil call is no-op"},
		{name: "does not panic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() {
				releaseHarnessAfterRunAbort(nil, "sess-nil")
			})
		})
	}
}

func TestAbortSessionHarness(t *testing.T) {
	tests := []struct {
		name  string
		setup func(sessionID string) *harness.Harness
	}{
		{
			name:  "missing session is no-op",
			setup: func(string) *harness.Harness { return nil },
		},
		{
			name: "aborts pooled harness loop",
			setup: func(sessionID string) *harness.Harness {
				blocking := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "blocked"})
				h := harness.New(harness.Options{
					AgentOptions: agent.Options{Model: blocking},
					ModelName:    "fake",
				})
				harnessPool.Store(sessionID, &pooledHarness{harness: h})
				return h
			},
		},
		{
			name: "invalid pool entry is no-op",
			setup: func(sessionID string) *harness.Harness {
				harnessPool.Store(sessionID, "not-a-harness")
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sessionID := "abort-" + tt.name
			t.Cleanup(func() { EvictHarnessPool(sessionID) })

			h := tt.setup(sessionID)
			if h == nil {
				assert.NotPanics(t, func() { AbortSessionHarness(sessionID) })
				return
			}

			runCtx, cancel := context.WithCancel(context.Background())
			_, err := h.Prompt(runCtx, agent.NewUserMessage("hello"))
			require.NoError(t, err)

			done := make(chan struct{})
			go func() {
				defer close(done)
				AbortSessionHarness(sessionID)
				cancel()
				releaseHarnessAfterRunAbort(h, sessionID)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("abort did not finish")
			}

			require.NoError(t, h.WaitIdle(context.Background()))
		})
	}
}
