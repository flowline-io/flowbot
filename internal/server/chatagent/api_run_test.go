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
			svc := NewService()
			switch tt.name {
			case "cancels bound run context":
				ctx, cancel := context.WithCancel(context.Background())
				svc.registerRunCancel("sess-cancel-run", cancel)
				t.Cleanup(func() { svc.unregisterRunCancel("sess-cancel-run") })
				svc.CancelSessionRun("sess-cancel-run")
				require.ErrorIs(t, ctx.Err(), context.Canceled)
			case "no-op without active run":
				assert.NotPanics(t, func() { svc.CancelSessionRun("sess-no-run") })
			case "aborts pooled harness without panic":
				sessionID := "sess-cancel-harness"
				h := harness.New(harness.Options{
					AgentOptions: agent.Options{Model: agentllm.NewFakeModel(agentllm.ResponseScript{Content: "ok"})},
					ModelName:    "fake",
				})
				svc.harnessPoolMap().Store(sessionID, &pooledHarness{harness: h})
				t.Cleanup(func() { svc.EvictHarnessPool(sessionID) })
				assert.NotPanics(t, func() { svc.CancelSessionRun(sessionID) })
			}
		})
	}
}

func TestResolveConfirmErrors(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Service) (string, string)
		approved bool
		wantErr  error
	}{
		{
			name: "missing session gate",
			setup: func(*Service) (string, string) {
				return "missing-session", "confirm-1"
			},
			wantErr: ErrConfirmNotFound,
		},
		{
			name: "wrong confirm id",
			setup: func(svc *Service) (string, string) {
				sessionID := "sess-wrong-id"
				pub := NewChannelPublisher(4)
				gate := NewConfirmGate(sessionID, pub, nil)
				require.NoError(t, svc.TrySetAPIRunState(sessionID, NewAPIRunState(pub, gate)))
				t.Cleanup(func() { svc.ClearAPIRunState(sessionID, nil) })
				return sessionID, "wrong-id"
			},
			wantErr: ErrConfirmNotFound,
		},
		{
			name: "nil run state rejected",
			setup: func(*Service) (string, string) {
				return "sess-nil-state", ""
			},
			wantErr: ErrConfirmNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
			sessionID, confirmID := tt.setup(svc)
			ok, err := svc.ResolveConfirm(sessionID, confirmID, tt.approved, "", "", ConfirmReasonDenied)
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
	svc := NewService()
	sessionID := "sess-force-clear"
	pub := NewChannelPublisher(4)
	gate := NewConfirmGate(sessionID, pub, nil)
	state := NewAPIRunState(pub, gate)
	require.NoError(t, svc.TrySetAPIRunState(sessionID, state))
	t.Cleanup(func() { svc.ClearAPIRunState(sessionID, nil) })

	svc.ClearAPIRunState(sessionID, nil)
	_, ok := svc.GetAPIRunState(sessionID)
	assert.False(t, ok)
}

func TestTrySetAPIRunState(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Service) string
		wantErr   bool
		wantClear bool
	}{
		{
			name: "registers first run",
			setup: func(*Service) string {
				return "sess-a"
			},
		},
		{
			name: "rejects concurrent run",
			setup: func(svc *Service) string {
				sessionID := "sess-b"
				pub := NewChannelPublisher(4)
				gate := NewConfirmGate(sessionID, pub, nil)
				require.NoError(t, svc.TrySetAPIRunState(sessionID, NewAPIRunState(pub, gate)))
				return sessionID
			},
			wantErr: true,
		},
		{
			name: "clear only matching state",
			setup: func(svc *Service) string {
				sessionID := "sess-c"
				pub := NewChannelPublisher(4)
				gate := NewConfirmGate(sessionID, pub, nil)
				state := NewAPIRunState(pub, gate)
				require.NoError(t, svc.TrySetAPIRunState(sessionID, state))
				svc.ClearAPIRunState(sessionID, state)
				return sessionID
			},
			wantClear: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
			sessionID := tt.setup(svc)
			t.Cleanup(func() { svc.ClearAPIRunState(sessionID, nil) })

			if tt.wantClear {
				_, ok := svc.GetAPIRunState(sessionID)
				assert.False(t, ok)
				return
			}

			pub := NewChannelPublisher(4)
			gate := NewConfirmGate(sessionID, pub, nil)
			state := NewAPIRunState(pub, gate)
			err := svc.TrySetAPIRunState(sessionID, state)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrRunInFlight)
				return
			}
			require.NoError(t, err)
			got, ok := svc.GetAPIRunState(sessionID)
			require.True(t, ok)
			assert.Equal(t, state, got)
		})
	}
}
