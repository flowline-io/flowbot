package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestClearConfirmState(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "returns empty fields"},
		{name: "idempotent clear"},
		{name: "safe after clear"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, tool, summary := ClearConfirmState()
			assert.Empty(t, id)
			assert.Empty(t, tool)
			assert.Empty(t, summary)
		})
	}
}

func TestHandleConfirmKey(t *testing.T) {
	tests := []struct {
		name     string
		start    int
		key      rune
		wantPick int
		wantOK   bool
	}{
		{name: "down selects deny", start: confirmChoiceApprove, key: tea.KeyDown, wantPick: confirmChoiceDeny, wantOK: true},
		{name: "up wraps to approve", start: confirmChoiceDeny, key: tea.KeyUp, wantPick: confirmChoiceApprove, wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.phase = PhaseConfirming
			m.confirmPick = tt.start
			m.pendingConfirmID = "confirm-1"
			ok, _ := m.handleConfirmKey(tea.KeyPressMsg{Code: tt.key})
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantPick, m.confirmPick)
		})
	}
}

func TestRenderConfirmPrompt(t *testing.T) {
	tests := []struct {
		name    string
		pick    int
		wantSub string
	}{
		{name: "shows approve selected", pick: confirmChoiceApprove, wantSub: "▸ Approve"},
		{name: "shows deny selected", pick: confirmChoiceDeny, wantSub: "▸ Deny"},
		{name: "shows navigation hint", pick: confirmChoiceApprove, wantSub: "↑↓ select"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.phase = PhaseConfirming
			m.confirmTool = "run_terminal"
			m.confirmSummary = "command: ls -all"
			m.confirmPick = tt.pick
			got := m.renderConfirmPrompt()
			assert.Contains(t, got, tt.wantSub)
			assert.Contains(t, got, "run_terminal")
			assert.Contains(t, got, "command: ls -all")
		})
	}
}

func TestConfirmChoiceLabel(t *testing.T) {
	tests := []struct {
		name string
		pick int
		want string
	}{
		{name: "approve label", pick: confirmChoiceApprove, want: "Approve"},
		{name: "deny label", pick: confirmChoiceDeny, want: "Deny"},
		{name: "invalid pick", pick: 9, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, confirmChoiceLabel(tt.pick))
		})
	}
}
