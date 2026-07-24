package chatagent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// stallModel blocks the first GenerateContent call until the test releases it, so
// callers can cancel a run while the harness is still busy without racing completion.
type stallModel struct {
	mu           sync.Mutex
	invoked      chan struct{}
	waitCanceled chan struct{}
	release      chan struct{}
	released     bool
}

func newStallModel() *stallModel {
	return &stallModel{
		invoked:      make(chan struct{}),
		waitCanceled: make(chan struct{}),
		release:      make(chan struct{}),
	}
}

func (m *stallModel) markCanceledWait() {
	close(m.waitCanceled)
}

func (m *stallModel) unblock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.released {
		return
	}
	m.released = true
	close(m.release)
}

func (m *stallModel) signalInvoked() {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.invoked:
	default:
		close(m.invoked)
	}
}

func (m *stallModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	resp, err := m.GenerateContent(ctx, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, prompt)}, options...)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", errors.New("stall model: empty response")
	}
	return resp.Choices[0].Content, nil
}

func (m *stallModel) GenerateContent(ctx context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
	m.signalInvoked()

	<-m.waitCanceled
	select {
	case <-m.release:
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{{Content: "ok", StopReason: "stop"}},
		}, nil
	case <-ctx.Done():
		<-m.release
		return nil, ctx.Err()
	}
}

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
		name      string
		sessionID string
	}{
		{name: "single abort drain", sessionID: "sess-abort-1"},
		{name: "multi abort drain", sessionID: "sess-abort-2"},
		{name: "follow-up prompt", sessionID: "sess-abort-3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewService()
			ctx, cancel := context.WithCancel(context.Background())
			stall := newStallModel()
			h := harness.New(harness.Options{
				AgentOptions: agent.Options{Model: stall},
				ModelName:    "fake",
			})

			_, err := h.Prompt(ctx, agent.NewUserMessage("hello"))
			require.NoError(t, err)
			<-stall.invoked

			idleErr := make(chan error, 1)
			go func() { idleErr <- h.WaitIdle(ctx) }()
			stall.markCanceledWait()
			cancel()

			require.ErrorIs(t, <-idleErr, context.Canceled)

			stall.unblock()
			svc.releaseHarnessAfterRunAbort(h, tt.sessionID)
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
			svc := NewService()
			assert.NotPanics(t, func() {
				svc.releaseHarnessAfterRunAbort(nil, "sess-nil")
			})
		})
	}
}

func TestAbortSessionHarness(t *testing.T) {
	tests := []struct {
		name  string
		setup func(svc *Service, sessionID string) *harness.Harness
	}{
		{
			name:  "missing session is no-op",
			setup: func(*Service, string) *harness.Harness { return nil },
		},
		{
			name: "aborts pooled harness loop",
			setup: func(svc *Service, sessionID string) *harness.Harness {
				blocking := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "blocked"})
				h := harness.New(harness.Options{
					AgentOptions: agent.Options{Model: blocking},
					ModelName:    "fake",
				})
				svc.harnessPoolMap().Store(sessionID, &pooledHarness{harness: h})
				return h
			},
		},
		{
			name: "invalid pool entry is no-op",
			setup: func(svc *Service, sessionID string) *harness.Harness {
				svc.harnessPoolMap().Store(sessionID, "not-a-harness")
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewService()
			sessionID := "abort-" + tt.name
			t.Cleanup(func() { svc.EvictHarnessPool(sessionID) })

			h := tt.setup(svc, sessionID)
			if h == nil {
				assert.NotPanics(t, func() { svc.AbortSessionHarness(sessionID) })
				return
			}

			runCtx, cancel := context.WithCancel(context.Background())
			_, err := h.Prompt(runCtx, agent.NewUserMessage("hello"))
			require.NoError(t, err)

			done := make(chan struct{})
			go func() {
				defer close(done)
				svc.AbortSessionHarness(sessionID)
				cancel()
				svc.releaseHarnessAfterRunAbort(h, sessionID)
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
