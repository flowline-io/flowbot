package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountPendingApprovalSessions(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, svc *Service) string
		wantDelta int
	}{
		{
			name: "idle does not add pending sessions",
			setup: func(t *testing.T, _ *Service) string {
				t.Helper()
				return ""
			},
			wantDelta: 0,
		},
		{
			name: "counts one waiting gate",
			setup: func(t *testing.T, svc *Service) string {
				t.Helper()
				sessionID := "sess-count-one"
				pub := NewChannelPublisher(8)
				gate := NewConfirmGate(sessionID, pub, nil)
				gate.timeout = 2 * time.Second
				state := NewAPIRunState(pub, gate)
				require.NoError(t, svc.TrySetAPIRunState(sessionID, state))
				t.Cleanup(func() { svc.ClearAPIRunState(sessionID, state) })
				go func() {
					_, _ = gate.Wait(context.Background(), hooks.ToolCallEvent{
						ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
						Args:     map[string]any{"command": "ls"},
					}, testEvalResult())
				}()
				waitConfirmEvent(t, pub)
				return sessionID
			},
			wantDelta: 1,
		},
		{
			name: "ignores running without pending confirm",
			setup: func(t *testing.T, svc *Service) string {
				t.Helper()
				sessionID := "sess-count-running"
				state := NewAPIRunState(NewChannelPublisher(4), NewConfirmGate(sessionID, nil, nil))
				require.NoError(t, svc.TrySetAPIRunState(sessionID, state))
				t.Cleanup(func() { svc.ClearAPIRunState(sessionID, state) })
				return sessionID
			},
			wantDelta: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
			before := svc.CountPendingApprovalSessions()
			sessionID := tt.setup(t, svc)
			assert.Equal(t, before+tt.wantDelta, svc.CountPendingApprovalSessions())
			if sessionID != "" && tt.wantDelta > 0 {
				assert.Contains(t, svc.ListSessionIDsByActivity(SessionActivityNeedsApproval), sessionID)
			}
		})
	}
}

func TestSessionActivity(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, svc *Service) string
		want  string
	}{
		{
			name: "idle session",
			setup: func(t *testing.T, _ *Service) string {
				t.Helper()
				return "sess-idle"
			},
			want: "",
		},
		{
			name: "running session",
			setup: func(t *testing.T, svc *Service) string {
				t.Helper()
				sessionID := "sess-running"
				state := NewAPIRunState(NewChannelPublisher(4), NewConfirmGate(sessionID, nil, nil))
				require.NoError(t, svc.TrySetAPIRunState(sessionID, state))
				t.Cleanup(func() { svc.ClearAPIRunState(sessionID, state) })
				return sessionID
			},
			want: SessionActivityRunning,
		},
		{
			name: "needs approval",
			setup: func(t *testing.T, svc *Service) string {
				t.Helper()
				sessionID := "sess-needs-approval"
				pub := NewChannelPublisher(8)
				gate := NewConfirmGate(sessionID, pub, nil)
				gate.timeout = 2 * time.Second
				state := NewAPIRunState(pub, gate)
				require.NoError(t, svc.TrySetAPIRunState(sessionID, state))
				t.Cleanup(func() { svc.ClearAPIRunState(sessionID, state) })
				go func() {
					_, _ = gate.Wait(context.Background(), hooks.ToolCallEvent{
						ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
						Args:     map[string]any{"command": "ls"},
					}, testEvalResult())
				}()
				waitConfirmEvent(t, pub)
				return sessionID
			},
			want: SessionActivityNeedsApproval,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
			sessionID := tt.setup(t, svc)
			assert.Equal(t, tt.want, svc.SessionActivity(sessionID))
			if tt.want == SessionActivityNeedsApproval {
				assert.Contains(t, svc.ListSessionIDsByActivity(SessionActivityNeedsApproval), sessionID)
				assert.NotContains(t, svc.ListSessionIDsByActivity(SessionActivityRunning), sessionID)
			}
			if tt.want == SessionActivityRunning {
				assert.Contains(t, svc.ListSessionIDsByActivity(SessionActivityRunning), sessionID)
			}
		})
	}
}

func TestConfirmGateIsWaiting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, gate *ConfirmGate, pub *ChannelPublisher)
	}{
		{
			name: "false before wait",
			run: func(t *testing.T, gate *ConfirmGate, _ *ChannelPublisher) {
				assert.False(t, gate.IsWaiting())
			},
		},
		{
			name: "true while waiting",
			run: func(t *testing.T, gate *ConfirmGate, pub *ChannelPublisher) {
				done := make(chan struct{})
				go func() {
					_, _ = gate.Wait(context.Background(), hooks.ToolCallEvent{
						ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
						Args:     map[string]any{"command": "ls"},
					}, testEvalResult())
					close(done)
				}()
				waitConfirmEvent(t, pub)
				assert.True(t, gate.IsWaiting())
				require.True(t, gate.Resolve(ConfirmResponse{
					Approved: true,
					Reason:   ConfirmReasonApproved,
					Mode:     ConfirmModeOnce,
				}))
				<-done
				assert.False(t, gate.IsWaiting())
			},
		},
		{
			name: "false after cancel",
			run: func(t *testing.T, gate *ConfirmGate, pub *ChannelPublisher) {
				done := make(chan struct{})
				go func() {
					_, _ = gate.Wait(context.Background(), hooks.ToolCallEvent{
						ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
						Args:     map[string]any{"command": "ls"},
					}, testEvalResult())
					close(done)
				}()
				waitConfirmEvent(t, pub)
				gate.Cancel()
				<-done
				assert.False(t, gate.IsWaiting())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := NewChannelPublisher(8)
			gate := NewConfirmGate("sess-waiting", pub, nil)
			gate.timeout = 2 * time.Second
			tt.run(t, gate, pub)
		})
	}
}
