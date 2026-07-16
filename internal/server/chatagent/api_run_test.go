package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIRunStatePublisher(t *testing.T) {
	tests := []struct {
		name  string
		state *APIRunState
		want  EventPublisher
	}{
		{name: "nil state", state: nil, want: nil},
		{name: "returns channel publisher", state: NewAPIRunState(NewChannelPublisher(2), nil), want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.Publisher()
			if tt.name == "returns channel publisher" {
				require.NotNil(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCancelSessionRun(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cancels bound run context"},
		{name: "no-op without active run"},
		{name: "aborts pooled harness without panic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "cancels bound run context":
				ctx, cancel := context.WithCancel(context.Background())
				registerRunCancel("sess-cancel-run", cancel)
				t.Cleanup(func() { unregisterRunCancel("sess-cancel-run") })
				CancelSessionRun("sess-cancel-run")
				require.ErrorIs(t, ctx.Err(), context.Canceled)
			case "no-op without active run":
				assert.NotPanics(t, func() { CancelSessionRun("sess-no-run") })
			case "aborts pooled harness without panic":
				sessionID := "sess-cancel-harness"
				h := harness.New(harness.Options{
					AgentOptions: agent.Options{Model: agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})},
					ModelName:    "fake",
				})
				harnessPool.Store(sessionID, &pooledHarness{harness: h})
				t.Cleanup(func() { EvictHarnessPool(sessionID) })
				assert.NotPanics(t, func() { CancelSessionRun(sessionID) })
			}
		})
	}
}

func TestResolveConfirmErrors(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (string, string)
		approved bool
		wantErr  error
	}{
		{
			name: "missing session gate",
			setup: func() (string, string) {
				return "missing-session", "confirm-1"
			},
			wantErr: ErrConfirmNotFound,
		},
		{
			name: "wrong confirm id",
			setup: func() (string, string) {
				sessionID := "sess-wrong-id"
				pub := NewChannelPublisher(4)
				gate := NewConfirmGate(sessionID, pub)
				require.NoError(t, TrySetAPIRunState(sessionID, NewAPIRunState(pub, gate)))
				t.Cleanup(func() { ClearAPIRunState(sessionID, nil) })
				return sessionID, "wrong-id"
			},
			wantErr: ErrConfirmNotFound,
		},
		{
			name: "nil run state rejected",
			setup: func() (string, string) {
				return "sess-nil-state", ""
			},
			wantErr: ErrConfirmNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID, confirmID := tt.setup()
			ok, err := ResolveConfirm(sessionID, confirmID, tt.approved, "", "", ConfirmReasonDenied)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.False(t, ok)
				return
			}
			require.NoError(t, err)
			assert.True(t, ok)
		})
	}
}

func TestClearAPIRunStateWithoutExpected(t *testing.T) {
	sessionID := "sess-force-clear"
	pub := NewChannelPublisher(4)
	gate := NewConfirmGate(sessionID, pub)
	state := NewAPIRunState(pub, gate)
	require.NoError(t, TrySetAPIRunState(sessionID, state))
	t.Cleanup(func() { ClearAPIRunState(sessionID, nil) })

	ClearAPIRunState(sessionID, nil)
	_, ok := GetAPIRunState(sessionID)
	assert.False(t, ok)
}

func TestTrySetAPIRunState(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() string
		wantErr   bool
		wantClear bool
	}{
		{
			name: "registers first run",
			setup: func() string {
				return "sess-a"
			},
		},
		{
			name: "rejects concurrent run",
			setup: func() string {
				sessionID := "sess-b"
				pub := NewChannelPublisher(4)
				gate := NewConfirmGate(sessionID, pub)
				require.NoError(t, TrySetAPIRunState(sessionID, NewAPIRunState(pub, gate)))
				return sessionID
			},
			wantErr: true,
		},
		{
			name: "clear only matching state",
			setup: func() string {
				sessionID := "sess-c"
				pub := NewChannelPublisher(4)
				gate := NewConfirmGate(sessionID, pub)
				state := NewAPIRunState(pub, gate)
				require.NoError(t, TrySetAPIRunState(sessionID, state))
				ClearAPIRunState(sessionID, state)
				return sessionID
			},
			wantClear: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID := tt.setup()
			t.Cleanup(func() { ClearAPIRunState(sessionID, nil) })

			if tt.wantClear {
				_, ok := GetAPIRunState(sessionID)
				assert.False(t, ok)
				return
			}

			pub := NewChannelPublisher(4)
			gate := NewConfirmGate(sessionID, pub)
			state := NewAPIRunState(pub, gate)
			err := TrySetAPIRunState(sessionID, state)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrRunInFlight)
				return
			}
			require.NoError(t, err)
			got, ok := GetAPIRunState(sessionID)
			require.True(t, ok)
			assert.Equal(t, state, got)
		})
	}
}
