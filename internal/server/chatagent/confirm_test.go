package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

			done := make(chan struct {
				approved bool
				err      error
			}, 1)
			go func() {
				approved, err := gate.Wait(context.Background(), hooks.ToolCallEvent{
					ToolCall: msg.ToolCallPart{Name: "run_terminal"},
					Args:     map[string]any{"command": "ls"},
				})
				done <- struct {
					approved bool
					err      error
				}{approved: approved, err: err}
			}()

			waitConfirmEvent(t, pub)

			require.True(t, gate.Resolve(tt.approved, tt.reason))
			waitResult := <-done
			require.NoError(t, waitResult.err)
			assert.Equal(t, tt.approved, waitResult.approved)
			assert.False(t, gate.Resolve(tt.approved, tt.reason))

			waitResolvedEvent(t, pub, tt.approved, string(tt.reason))
		})
	}
}

func TestConfirmGateTimeout(t *testing.T) {
	pub := NewChannelPublisher(4)
	gate := NewConfirmGate("sess-1", pub)
	gate.timeout = 20 * time.Millisecond

	done := make(chan struct {
		approved bool
		err      error
	}, 1)
	go func() {
		approved, err := gate.Wait(context.Background(), hooks.ToolCallEvent{
			ToolCall: msg.ToolCallPart{Name: "write_file"},
			Args:     map[string]any{"path": "a.txt"},
		})
		done <- struct {
			approved bool
			err      error
		}{approved: approved, err: err}
	}()

	waitConfirmEvent(t, pub)
	waitResult := <-done
	require.NoError(t, waitResult.err)
	assert.False(t, waitResult.approved)
	waitResolvedEvent(t, pub, false, string(ConfirmReasonTimeout))
}

func TestConfirmGateMultipleTools(t *testing.T) {
	pub := NewChannelPublisher(8)
	gate := NewConfirmGate("sess-1", pub)
	gate.timeout = 2 * time.Second

	done1 := make(chan struct {
		approved bool
		err      error
	}, 1)
	go func() {
		approved, err := gate.Wait(context.Background(), hooks.ToolCallEvent{
			ToolCall: msg.ToolCallPart{Name: "run_terminal"},
			Args:     map[string]any{"command": "ls"},
		})
		done1 <- struct {
			approved bool
			err      error
		}{approved: approved, err: err}
	}()

	waitConfirmEvent(t, pub)
	id1 := gate.ID()
	require.True(t, gate.Resolve(true, ConfirmReasonApproved))
	waitResult1 := <-done1
	require.NoError(t, waitResult1.err)
	assert.True(t, waitResult1.approved)
	waitResolvedEvent(t, pub, true, string(ConfirmReasonApproved))

	done2 := make(chan struct {
		approved bool
		err      error
	}, 1)
	go func() {
		approved, err := gate.Wait(context.Background(), hooks.ToolCallEvent{
			ToolCall: msg.ToolCallPart{Name: "write_file"},
			Args:     map[string]any{"path": "a.txt"},
		})
		done2 <- struct {
			approved bool
			err      error
		}{approved: approved, err: err}
	}()

	waitConfirmEvent(t, pub)
	id2 := gate.ID()
	assert.NotEqual(t, id1, id2)
	require.True(t, gate.Resolve(true, ConfirmReasonApproved))
	waitResult2 := <-done2
	require.NoError(t, waitResult2.err)
	assert.True(t, waitResult2.approved)
	waitResolvedEvent(t, pub, true, string(ConfirmReasonApproved))
}

func TestResolveConfirmAPI(t *testing.T) {
	pub := NewChannelPublisher(4)
	gate := NewConfirmGate("sess-2", pub)
	state := NewAPIRunState(pub, gate)
	require.NoError(t, TrySetAPIRunState("sess-2", state))
	t.Cleanup(func() { ClearAPIRunState("sess-2", nil) })

	ok, err := ResolveConfirm("sess-2", gate.ID(), true, ConfirmReasonApproved)
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = ResolveConfirm("sess-2", gate.ID(), true, ConfirmReasonApproved)
	assert.ErrorIs(t, err, ErrConfirmResolved)
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
