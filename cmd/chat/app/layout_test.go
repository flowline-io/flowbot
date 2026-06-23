package app

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestRenderCompactHeader(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		sessionTitle string
		wantSub      []string
	}{
		{name: "includes title and rule", width: 80, wantSub: []string{"Flowbot Agent", "─"}},
		{name: "narrow width", width: 30, wantSub: []string{"Flowbot Agent"}},
		{name: "minimum width", width: 14, wantSub: []string{"Flowbot Agent"}},
		{name: "uses session title when set", width: 80, sessionTitle: "Redis setup", wantSub: []string{"Flowbot Agent", "Redis setup", "─"}},
	}
	styles := NewStyles()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(renderCompactHeader(tt.width, tt.sessionTitle, &styles))
			for _, want := range tt.wantSub {
				assert.Contains(t, got, want)
			}
		})
	}
}

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

func TestRenderTopSectionShowsTitleOnStartup(t *testing.T) {
	tests := []struct {
		name         string
		sessionTitle string
		splash       bool
		wantSub      []string
		notWant      string
	}{
		{
			name:         "splash startup shows title in compact header",
			sessionTitle: "Redis setup",
			splash:       true,
			wantSub:      []string{"Flowbot Agent", "Redis setup"},
		},
		{
			name:         "active transcript keeps compact header",
			sessionTitle: "Deploy flowbot",
			splash:       false,
			wantSub:      []string{"Flowbot Agent", "Deploy flowbot"},
		},
		{
			name:    "splash without title keeps ascii banner",
			splash:  true,
			wantSub: []string{"Flowbot Agent", "Available Tools"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 100
			m.height = 40
			m.info = &client.ChatAgentInfo{
				Version:   "1.0.0",
				ChatModel: "gpt-test",
				Provider:  "openai",
			}
			m.sessionTitle = tt.sessionTitle
			m.splashVisible = tt.splash
			if !tt.splash {
				m.transcript.WriteString("hello")
			}
			got := stripANSI(m.renderTopSection())
			for _, want := range tt.wantSub {
				assert.Contains(t, got, want)
			}
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
