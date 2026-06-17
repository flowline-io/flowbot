package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/flowline-io/flowbot/pkg/client"
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
		always   bool
		wantPick int
		wantOK   bool
	}{
		{name: "down selects next", start: 0, key: tea.KeyDown, always: true, wantPick: 1, wantOK: true},
		{name: "up wraps to last", start: 0, key: tea.KeyUp, always: true, wantPick: 2, wantOK: true},
		{name: "two choices without always", start: 0, key: tea.KeyDown, always: false, wantPick: 1, wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.phase = PhaseConfirming
			m.confirmPick = tt.start
			m.confirmSuggestAlways = tt.always
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
		always  bool
		wantSub string
	}{
		{name: "shows once selected", pick: 0, wantSub: "▸ Approve once"},
		{name: "shows deny selected", pick: 2, always: true, wantSub: "▸ Deny"},
		{name: "shows navigation hint", pick: 0, wantSub: "↑↓ select"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.phase = PhaseConfirming
			m.confirmTool = "run_terminal"
			m.confirmSummary = "command: ls -all"
			m.confirmSuggestAlways = tt.always
			m.confirmPick = tt.pick
			got := m.renderConfirmPrompt()
			assert.Contains(t, got, tt.wantSub)
			assert.Contains(t, got, "run_terminal")
			assert.Contains(t, got, "command: ls -all")
		})
	}
}

func TestConfirmChoiceLabel(t *testing.T) {
	choices := []confirmChoice{
		{label: "Approve once"},
		{label: "Deny"},
	}
	tests := []struct {
		name string
		pick int
		want string
	}{
		{name: "once label", pick: 0, want: "Approve once"},
		{name: "deny label", pick: 1, want: "Deny"},
		{name: "invalid pick", pick: 9, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, confirmChoiceLabel(tt.pick, choices))
		})
	}
}

func TestSubmitConfirmChoiceUsesMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/confirm") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	m := NewModel(client.NewClient(srv.URL, "token"), "default")
	m.sessionID = "sess-1"
	m.phase = PhaseConfirming
	m.pendingConfirmID = "c1"
	m.confirmSuggestedPattern = "git status*"
	m.confirmSuggestAlways = true
	cmd := m.submitConfirmChoice(confirmChoice{approved: true, mode: client.ConfirmModeAlways})
	assert.NotNil(t, cmd)
	assert.Equal(t, PhaseStreaming, m.phase)
}
