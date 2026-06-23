package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestFormatSessionRow(t *testing.T) {
	styles := NewStyles()
	now := time.Now().Add(-30 * time.Minute)
	summary := client.ChatSessionSummary{
		SessionID: "abcdefghijklmnop",
		UpdatedAt: now,
	}

	tests := []struct {
		name     string
		current  string
		selected bool
		summary  client.ChatSessionSummary
		wantSub  string
	}{
		{name: "selected row uses marker", current: "", selected: true, wantSub: "▸ abcdefghijklmnop"},
		{name: "current session label", current: "abcdefghijklmnop", selected: false, wantSub: "current"},
		{name: "plain row shows full id", current: "other-session", selected: false, wantSub: "abcdefghijklmnop"},
		{
			name:     "plan session shows plan label",
			current:  "",
			selected: false,
			summary:  client.ChatSessionSummary{SessionID: "sess-plan", Mode: sessionModePlan, UpdatedAt: now},
			wantSub:  " · plan",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := tt.summary
			if row.SessionID == "" {
				row = summary
			}
			got := FormatSessionRow(row, tt.current, tt.selected, &styles)
			assert.Contains(t, got, tt.wantSub)
		})
	}
}

func TestFormatSessionUpdatedAt(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{name: "zero time", at: time.Time{}, want: "unknown"},
		{name: "recent update", at: time.Now().Add(-2 * time.Minute), want: "ago"},
		{name: "older update", at: time.Now().Add(-26 * time.Hour), want: "202"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSessionUpdatedAt(tt.at)
			assert.Contains(t, got, tt.want)
		})
	}
}

func TestSessionPickerInitialPick(t *testing.T) {
	sessions := []client.ChatSessionSummary{
		{SessionID: "sess-a"},
		{SessionID: "sess-b"},
		{SessionID: "sess-c"},
	}

	tests := []struct {
		name    string
		current string
		want    int
	}{
		{name: "finds current session", current: "sess-b", want: 1},
		{name: "defaults to first", current: "missing", want: 0},
		{name: "empty current defaults to first", current: "", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sessionPickerInitialPick(sessions, tt.current))
		})
	}
}

func TestHandleSessionPickKey(t *testing.T) {
	tests := []struct {
		name     string
		start    int
		key      tea.KeyPressMsg
		wantPick int
		wantIdle bool
	}{
		{name: "down moves selection", start: 0, key: tea.KeyPressMsg{Code: tea.KeyDown}, wantPick: 1},
		{name: "up wraps selection", start: 0, key: tea.KeyPressMsg{Code: tea.KeyUp}, wantPick: 2},
		{name: "escape clears picker", start: 1, key: tea.KeyPressMsg{Code: tea.KeyEscape}, wantIdle: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.phase = PhaseSessionPick
			m.picker.list = []client.ChatSessionSummary{
				{SessionID: "sess-a"},
				{SessionID: "sess-b"},
				{SessionID: "sess-c"},
			}
			m.picker.pick = tt.start

			handled, _ := m.handleSessionPickKey(tt.key)
			assert.True(t, handled)
			if tt.wantIdle {
				assert.Equal(t, PhaseIdle, m.phase)
				assert.Nil(t, m.picker.list)
				return
			}
			assert.Equal(t, tt.wantPick, m.picker.pick)
		})
	}
}

func TestSubmitSessionPick(t *testing.T) {
	tests := []struct {
		name         string
		current      string
		pick         int
		wantID       string
		wantSwitch   bool
		wantPlanMode bool
		sessions     []client.ChatSessionSummary
	}{
		{name: "same session does not switch", current: "sess-b", pick: 1, wantID: "sess-b", wantSwitch: false, wantPlanMode: false},
		{
			name:         "switch clears plan when target session is normal",
			current:      "sess-a",
			pick:         2,
			wantID:       "sess-c",
			wantSwitch:   true,
			wantPlanMode: false,
			sessions: []client.ChatSessionSummary{
				{SessionID: "sess-a", Mode: sessionModePlan},
				{SessionID: "sess-b"},
				{SessionID: "sess-c", Mode: "normal"},
			},
		},
		{
			name:         "switch shows plan when target session is plan",
			current:      "sess-a",
			pick:         1,
			wantID:       "sess-b",
			wantSwitch:   true,
			wantPlanMode: true,
			sessions: []client.ChatSessionSummary{
				{SessionID: "sess-a", Mode: "normal"},
				{SessionID: "sess-b", Mode: sessionModePlan},
				{SessionID: "sess-c"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			m := NewModel(nil, "default")
			m.phase = PhaseSessionPick
			m.sessionID = tt.current
			if tt.name == "switch clears plan when target session is normal" {
				m.applySessionModeDisplay(sessionModePlan)
			}
			sessions := tt.sessions
			if len(sessions) == 0 {
				sessions = []client.ChatSessionSummary{
					{SessionID: "sess-a"},
					{SessionID: "sess-b"},
					{SessionID: "sess-c"},
				}
			}
			m.picker.list = sessions
			m.picker.pick = tt.pick

			_ = m.submitSessionPick()
			assert.Equal(t, PhaseIdle, m.phase)
			assert.Equal(t, tt.wantID, m.sessionID)
			assert.Equal(t, tt.wantPlanMode, m.status.PlanMode)
			if tt.wantSwitch && tt.sessions == nil {
				assert.Equal(t, "Switched session", m.hint)
			}
		})
	}
}

func TestUpdateSessionModeLoad(t *testing.T) {
	tests := []struct {
		name         string
		currentID    string
		currentMode  string
		msg          sessionModeLoadMsg
		wantPlanMode bool
		wantHintSub  string
		wantStale    bool
	}{
		{
			name:         "loads normal mode after switch from plan session",
			currentID:    "sess-b",
			currentMode:  sessionModePlan,
			msg:          sessionModeLoadMsg{sessionID: "sess-b", mode: "normal"},
			wantPlanMode: false,
			wantHintSub:  "/help",
		},
		{
			name:         "loads plan mode for switched session",
			currentID:    "sess-c",
			currentMode:  "normal",
			msg:          sessionModeLoadMsg{sessionID: "sess-c", mode: sessionModePlan},
			wantPlanMode: true,
			wantHintSub:  "Plan mode",
		},
		{
			name:         "ignores stale mode response",
			currentID:    "sess-b",
			currentMode:  sessionModePlan,
			msg:          sessionModeLoadMsg{sessionID: "sess-a", mode: "normal"},
			wantPlanMode: true,
			wantStale:    true,
		},
		{
			name:         "load error clears stale plan badge",
			currentID:    "sess-b",
			currentMode:  sessionModePlan,
			msg:          sessionModeLoadMsg{sessionID: "sess-b", err: "network error"},
			wantPlanMode: false,
			wantHintSub:  "network error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 24
			m.sessionID = tt.currentID
			m.applySessionModeDisplay(tt.currentMode)

			updated, _ := m.updateSessionModeLoad(tt.msg)
			if tt.wantStale {
				assert.True(t, updated.status.PlanMode)
				return
			}
			assert.Equal(t, tt.wantPlanMode, updated.status.PlanMode)
			assert.Contains(t, updated.hint, tt.wantHintSub)
		})
	}
}

func TestUpdateSessionsList(t *testing.T) {
	tests := []struct {
		name      string
		msg       sessionsListMsg
		wantPhase RunPhase
		wantHint  string
	}{
		{
			name:     "error sets hint",
			msg:      sessionsListMsg{err: "boom"},
			wantHint: "boom",
		},
		{
			name:     "empty list appends system message",
			msg:      sessionsListMsg{sessions: nil},
			wantHint: defaultHint(),
		},
		{
			name: "sessions open picker",
			msg: sessionsListMsg{sessions: []client.ChatSessionSummary{
				{SessionID: "sess-a"},
			}},
			wantPhase: PhaseSessionPick,
			wantHint:  "Select a session to switch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			updated, _ := m.updateSessionsList(tt.msg)
			if tt.wantPhase != 0 {
				assert.Equal(t, tt.wantPhase, updated.phase)
			}
			if tt.msg.err != "" || tt.wantPhase == PhaseSessionPick {
				assert.Equal(t, tt.wantHint, updated.hint)
			}
			if len(tt.msg.sessions) == 0 && tt.msg.err == "" {
				assert.Contains(t, updated.transcript.String(), "No sessions found")
			}
		})
	}
}
