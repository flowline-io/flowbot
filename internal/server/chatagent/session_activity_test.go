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

func TestSessionActivity(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		want    string
		cleanup func(sessionID string)
	}{
		{
			name: "idle session",
			setup: func(t *testing.T) string {
				t.Helper()
				return "sess-idle"
			},
			want: "",
		},
		{
			name: "running via active api run",
			setup: func(t *testing.T) string {
				t.Helper()
				sessionID := "sess-running"
				state := NewAPIRunState(NewChannelPublisher(4), NewConfirmGate(sessionID, nil))
				require.NoError(t, TrySetAPIRunState(sessionID, state))
				t.Cleanup(func() { ClearAPIRunState(sessionID, state) })
				return sessionID
			},
			want: SessionActivityRunning,
		},
		{
			name: "needs approval while waiting",
			setup: func(t *testing.T) string {
				t.Helper()
				sessionID := "sess-need-approval"
				pub := NewChannelPublisher(8)
				gate := NewConfirmGate(sessionID, pub)
				gate.timeout = 2 * time.Second
				state := NewAPIRunState(pub, gate)
				require.NoError(t, TrySetAPIRunState(sessionID, state))
				t.Cleanup(func() { ClearAPIRunState(sessionID, state) })

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
			sessionID := tt.setup(t)
			assert.Equal(t, tt.want, SessionActivity(sessionID))
			if tt.want == SessionActivityNeedsApproval {
				assert.Contains(t, ListSessionIDsByActivity(SessionActivityNeedsApproval), sessionID)
				assert.NotContains(t, ListSessionIDsByActivity(SessionActivityRunning), sessionID)
			}
			if tt.want == SessionActivityRunning {
				assert.Contains(t, ListSessionIDsByActivity(SessionActivityRunning), sessionID)
			}
		})
	}
}

func TestConfirmGateIsWaiting(t *testing.T) {
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
			name: "true while waiting then false after resolve",
			run: func(t *testing.T, gate *ConfirmGate, pub *ChannelPublisher) {
				done := make(chan struct{})
				go func() {
					defer close(done)
					_, _ = gate.Wait(context.Background(), hooks.ToolCallEvent{
						ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
						Args:     map[string]any{"command": "echo"},
					}, testEvalResult())
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
					defer close(done)
					_, _ = gate.Wait(context.Background(), hooks.ToolCallEvent{
						ToolCall: msg.ToolCallPart{Name: permission.ToolWriteFile},
						Args:     map[string]any{"path": "a.txt"},
					}, permission.Result{Action: permission.ActionAsk, PermissionKey: "edit", Pattern: "a.txt"})
				}()
				waitConfirmEvent(t, pub)
				assert.True(t, gate.IsWaiting())
				gate.Cancel()
				<-done
				assert.False(t, gate.IsWaiting())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := NewChannelPublisher(8)
			gate := NewConfirmGate("sess-waiting", pub)
			gate.timeout = 2 * time.Second
			tt.run(t, gate, pub)
		})
	}
}
