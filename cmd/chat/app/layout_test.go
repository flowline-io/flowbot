package app

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestFooterHeight(t *testing.T) {
	tests := []struct {
		name        string
		hint        string
		confirm     bool
		sessionPick bool
		wantAtLeast int
	}{
		{name: "idle defaults", wantAtLeast: 4},
		{name: "with hint line", hint: "/help", wantAtLeast: 5},
		{name: "confirming adds rows", hint: "approve", confirm: true, wantAtLeast: 7},
		{name: "session picker adds rows", sessionPick: true, wantAtLeast: 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.hint = tt.hint
			if tt.confirm {
				m.phase = PhaseConfirming
			}
			if tt.sessionPick {
				m.phase = PhaseSessionPick
				m.picker.list = []client.ChatSessionSummary{
					{SessionID: "sess-a", UpdatedAt: time.Now()},
					{SessionID: "sess-b", UpdatedAt: time.Now()},
				}
			}
			assert.GreaterOrEqual(t, m.footerHeight(), tt.wantAtLeast)
		})
	}
}

func TestSyncLayoutReservesFooter(t *testing.T) {
	tests := []struct {
		name   string
		height int
	}{
		{name: "standard terminal", height: 40},
		{name: "short terminal", height: 24},
		{name: "tall terminal", height: 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 100
			m.height = tt.height
			m.appendSystem(SlashHelp())
			m.syncLayout()
			footerH := m.footerHeight()
			headerH := lipglossHeight(m.renderTopSection())
			assert.LessOrEqual(t, m.viewport.Height()+headerH+footerH, tt.height+3)
		})
	}
}

func lipglossHeight(s string) int {
	return lenSplitLines(s)
}

func lenSplitLines(s string) int {
	if s == "" {
		return 0
	}
	n := 1
	for _, r := range s {
		if r == '\n' {
			n++
		}
	}
	return n
}
