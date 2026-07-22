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

func testEvalResult() permission.Result {
	return permission.Result{
		Action:        permission.ActionAsk,
		PermissionKey: "bash",
		Pattern:       "ls",
	}
}

func TestConfirmGateResolve(t *testing.T) {
	tests := []struct {
		name      string
		approved  bool
		reason    ConfirmReason
		wantBlock bool
	}{
		{name: "approved", approved: true, reason: ConfirmReasonApproved, wantBlock: false},
		{name: "denied", approved: false, reason: ConfirmReasonDenied, wantBlock: true},
		{name: "timeout reason", approved: false, reason: ConfirmReasonTimeout, wantBlock: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := NewChannelPublisher(8)
			gate := NewConfirmGate("sess-1", pub)
			gate.timeout = 2 * time.Second

			done := make(chan ConfirmResponse, 1)
			go func() {
				resp, err := gate.Wait(context.Background(), hooks.ToolCallEvent{
					ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
					Args:     map[string]any{"command": "ls"},
				}, testEvalResult())
				if err != nil {
					done <- ConfirmResponse{}
					return
				}
				done <- resp
			}()

			waitConfirmEvent(t, pub)

			mode := ConfirmModeReject
			if tt.approved {
				mode = ConfirmModeOnce
			}
			require.True(t, gate.Resolve(ConfirmResponse{Approved: tt.approved, Reason: tt.reason, Mode: mode}))
			waitResult := <-done
			assert.Equal(t, tt.approved, waitResult.Approved)
			assert.False(t, gate.Resolve(ConfirmResponse{Approved: tt.approved, Reason: tt.reason, Mode: mode}))

			waitResolvedEvent(t, pub, tt.approved, string(tt.reason))
		})
	}
}

func TestConfirmGateTimeout(t *testing.T) {
	pub := NewChannelPublisher(4)
	gate := NewConfirmGate("sess-1", pub)
	gate.timeout = 20 * time.Millisecond

	done := make(chan ConfirmResponse, 1)
	go func() {
		resp, err := gate.Wait(context.Background(), hooks.ToolCallEvent{
			ToolCall: msg.ToolCallPart{Name: permission.ToolWriteFile},
			Args:     map[string]any{"path": "a.txt"},
		}, permission.Result{Action: permission.ActionAsk, PermissionKey: "edit", Pattern: "a.txt"})
		if err != nil {
			done <- ConfirmResponse{}
			return
		}
		done <- resp
	}()

	waitConfirmEvent(t, pub)
	waitResult := <-done
	assert.False(t, waitResult.Approved)
	waitResolvedEvent(t, pub, false, string(ConfirmReasonTimeout))
}

func TestConfirmGateMultipleTools(t *testing.T) {
	pub := NewChannelPublisher(8)
	gate := NewConfirmGate("sess-1", pub)
	gate.timeout = 2 * time.Second

	done1 := make(chan ConfirmResponse, 1)
	go func() {
		resp, _ := gate.Wait(context.Background(), hooks.ToolCallEvent{
			ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
			Args:     map[string]any{"command": "ls"},
		}, testEvalResult())
		done1 <- resp
	}()

	waitConfirmEvent(t, pub)
	id1 := gate.ID()
	require.True(t, gate.Resolve(ConfirmResponse{Approved: true, Reason: ConfirmReasonApproved, Mode: ConfirmModeOnce}))
	waitResult1 := <-done1
	assert.True(t, waitResult1.Approved)
	waitResolvedEvent(t, pub, true, string(ConfirmReasonApproved))

	done2 := make(chan ConfirmResponse, 1)
	go func() {
		resp, _ := gate.Wait(context.Background(), hooks.ToolCallEvent{
			ToolCall: msg.ToolCallPart{Name: permission.ToolWriteFile},
			Args:     map[string]any{"path": "a.txt"},
		}, permission.Result{Action: permission.ActionAsk, PermissionKey: "edit", Pattern: "a.txt"})
		done2 <- resp
	}()

	waitConfirmEvent(t, pub)
	id2 := gate.ID()
	assert.NotEqual(t, id1, id2)
	require.True(t, gate.Resolve(ConfirmResponse{Approved: true, Reason: ConfirmReasonApproved, Mode: ConfirmModeOnce}))
	waitResult2 := <-done2
	assert.True(t, waitResult2.Approved)
	waitResolvedEvent(t, pub, true, string(ConfirmReasonApproved))
}

func TestResolveConfirmAPI(t *testing.T) {
	pub := NewChannelPublisher(4)
	gate := NewConfirmGate("sess-2", pub)
	state := NewAPIRunState(pub, gate)
	require.NoError(t, TrySetAPIRunState("sess-2", state))
	t.Cleanup(func() { ClearAPIRunState("sess-2", nil) })

	ok, err := ResolveConfirm("sess-2", gate.ID(), true, ConfirmModeOnce, "", ConfirmReasonApproved)
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = ResolveConfirm("sess-2", gate.ID(), true, ConfirmModeOnce, "", ConfirmReasonApproved)
	assert.ErrorIs(t, err, ErrConfirmResolved)
}

func TestPrematureClearAPIRunStateBreaksConfirm(t *testing.T) {
	pub := NewChannelPublisher(8)
	gate := NewConfirmGate("sess-premature", pub)
	gate.timeout = 2 * time.Second
	state := NewAPIRunState(pub, gate)
	require.NoError(t, TrySetAPIRunState("sess-premature", state))
	t.Cleanup(func() { ClearAPIRunState("sess-premature", nil) })

	go func() {
		_, _ = gate.Wait(context.Background(), hooks.ToolCallEvent{
			ToolCall: msg.ToolCallPart{Name: permission.ToolRunTerminal},
			Args:     map[string]any{"command": "ls"},
		}, testEvalResult())
	}()

	waitConfirmEvent(t, pub)
	confirmID := gate.ID()

	ClearAPIRunState("sess-premature", state)

	_, err := ResolveConfirm("sess-premature", confirmID, true, ConfirmModeOnce, "", ConfirmReasonApproved)
	assert.ErrorIs(t, err, ErrConfirmNotFound)
}

func TestAlwaysGrantPattern(t *testing.T) {
	tests := []struct {
		name          string
		eval          permission.Result
		clientPattern string
		wantPattern   string
		wantOK        bool
	}{
		{
			name:        "uses suggested when client empty",
			eval:        permission.Result{SuggestAlways: true, SuggestedPattern: "git status*"},
			wantPattern: "git status*",
			wantOK:      true,
		},
		{
			name:          "rejects broader client pattern",
			eval:          permission.Result{SuggestAlways: true, SuggestedPattern: "git status*"},
			clientPattern: "git *",
			wantOK:        false,
		},
		{
			name:          "rejects when always not suggested",
			eval:          permission.Result{SuggestAlways: false, SuggestedPattern: "git status*"},
			clientPattern: "git status*",
			wantOK:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, ok := alwaysGrantPattern(tt.eval, tt.clientPattern)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantPattern, pattern)
			}
		})
	}
}

func TestConfirmGatePendingEvent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T, gate *ConfirmGate, pub *ChannelPublisher)
	}{
		{
			name: "unavailable before wait",
			run: func(t *testing.T, gate *ConfirmGate, _ *ChannelPublisher) {
				_, ok := gate.PendingEvent()
				assert.False(t, ok)
			},
		},
		{
			name: "available while waiting",
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
				ev, ok := gate.PendingEvent()
				require.True(t, ok)
				assert.Equal(t, EventTypeConfirm, ev.Type)
				assert.Equal(t, permission.ToolRunTerminal, ev.Tool)
				assert.NotEmpty(t, ev.ID)
				require.True(t, gate.Resolve(ConfirmResponse{
					Approved: true,
					Reason:   ConfirmReasonApproved,
					Mode:     ConfirmModeOnce,
				}))
				<-done
				_, ok = gate.PendingEvent()
				assert.False(t, ok)
			},
		},
		{
			name: "lookup pending by session id",
			run: func(t *testing.T, gate *ConfirmGate, pub *ChannelPublisher) {
				sessionID := "sess-lookup-pending"
				state := NewAPIRunState(pub, gate)
				require.NoError(t, TrySetAPIRunState(sessionID, state))
				t.Cleanup(func() { ClearAPIRunState(sessionID, state) })

				done := make(chan struct{})
				go func() {
					_, _ = gate.Wait(context.Background(), hooks.ToolCallEvent{
						ToolCall: msg.ToolCallPart{Name: permission.ToolWriteFile},
						Args:     map[string]any{"path": "a.txt"},
					}, permission.Result{Action: permission.ActionAsk, PermissionKey: "edit", Pattern: "a.txt"})
					close(done)
				}()
				waitConfirmEvent(t, pub)
				ev, ok := LookupPendingConfirm(sessionID)
				require.True(t, ok)
				assert.Equal(t, permission.ToolWriteFile, ev.Tool)
				require.True(t, gate.Resolve(ConfirmResponse{
					Approved: false,
					Reason:   ConfirmReasonDenied,
					Mode:     ConfirmModeReject,
				}))
				<-done
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := NewChannelPublisher(8)
			gate := NewConfirmGate("sess-pending-event", pub)
			gate.timeout = 2 * time.Second
			tt.run(t, gate, pub)
		})
	}
}

func waitConfirmEvent(t *testing.T, pub *ChannelPublisher) {
	t.Helper()
	select {
	case ev := <-pub.Events():
		assert.Equal(t, EventTypeConfirm, ev.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for confirm event")
	}
}

func waitResolvedEvent(t *testing.T, pub *ChannelPublisher, approved bool, reason string) {
	t.Helper()
	select {
	case ev := <-pub.Events():
		assert.Equal(t, EventTypeConfirmResolved, ev.Type)
		assert.Equal(t, approved, ev.Approved)
		assert.Equal(t, reason, ev.Reason)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for confirm_resolved event")
	}
}
