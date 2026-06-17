package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/flowline-io/flowbot/pkg/client"
)

const maxSessionPickerLines = 12

// sessionsListMsg reports the result of an async /sessions command.
type sessionsListMsg struct {
	sessions []client.ChatSessionSummary
	err      string
}

// FormatSessionRow renders one session entry for the picker.
func FormatSessionRow(summary client.ChatSessionSummary, currentID string, selected bool, styles *Styles) string {
	label := summary.SessionID
	if summary.SessionID == currentID {
		label += " · current"
	}
	label += " · " + formatSessionUpdatedAt(summary.UpdatedAt)

	prefix := "  "
	if selected {
		return styles.InputPrompt.Render("▸ " + label)
	}
	return styles.Hint.Render(prefix + label)
}

func formatSessionUpdatedAt(updatedAt time.Time) string {
	if updatedAt.IsZero() {
		return "unknown"
	}
	elapsed := time.Since(updatedAt)
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	default:
		return updatedAt.Format("2006-01-02")
	}
}

func (m *Model) renderSessionPicker() string {
	if m.phase != PhaseSessionPick || len(m.sessionList) == 0 {
		return ""
	}
	var body strings.Builder
	writeBuilder(&body, "Sessions")
	writeBuilder(&body, "\n")
	limit := min(len(m.sessionList), maxSessionPickerLines)
	for i := range limit {
		writeBuilder(&body, FormatSessionRow(m.sessionList[i], m.sessionID, i == m.sessionPick, &m.styles))
		writeBuilder(&body, "\n")
	}
	if len(m.sessionList) > maxSessionPickerLines {
		writeBuilder(&body, m.styles.Hint.Render(fmt.Sprintf("  … and %d more", len(m.sessionList)-maxSessionPickerLines)))
		writeBuilder(&body, "\n")
	}
	writeBuilder(&body, m.styles.Hint.Render("  ↑↓ select · Enter switch · Esc cancel · Ctrl+C cancel"))
	return m.styles.ConfirmBox.Render(body.String())
}

func (m *Model) sessionPickerFooterHeight() int {
	if m.phase != PhaseSessionPick {
		return 0
	}
	lines := 2
	visible := len(m.sessionList)
	if visible > maxSessionPickerLines {
		visible = maxSessionPickerLines + 1
	}
	lines += visible
	return lines + 2
}

func (m *Model) handleSessionPickKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.phase != PhaseSessionPick {
		return false, nil
	}
	switch msg.Key().Code {
	case tea.KeyUp:
		if len(m.sessionList) == 0 {
			return true, nil
		}
		m.sessionPick = (m.sessionPick + len(m.sessionList) - 1) % len(m.sessionList)
		return true, nil
	case tea.KeyDown:
		if len(m.sessionList) == 0 {
			return true, nil
		}
		m.sessionPick = (m.sessionPick + 1) % len(m.sessionList)
		return true, nil
	case tea.KeyEnter, tea.KeyTab:
		return true, m.submitSessionPick()
	case tea.KeyEscape:
		m.clearSessionPick()
		m.phase = PhaseIdle
		m.hint = defaultHint()
		return true, m.focusInputCmd()
	}
	return false, nil
}

func (m *Model) clearSessionPick() {
	m.sessionList = nil
	m.sessionPick = 0
}

func (m *Model) applySessionSwitch(id string) {
	m.sessionID = id
	if hint := sessionCacheHint(SaveSessionID(m.profile, id)); hint != "" {
		m.hint = hint
	} else {
		m.hint = "Switched session"
	}
	m.transcript.Reset()
	m.streamOverlay.Reset()
	m.messageCount = 0
	m.resetSessionUsage()
	m.splashVisible = false
	m.syncViewport()
}

func (m *Model) submitSessionPick() tea.Cmd {
	if len(m.sessionList) == 0 || m.sessionPick < 0 || m.sessionPick >= len(m.sessionList) {
		m.clearSessionPick()
		m.phase = PhaseIdle
		return m.focusInputCmd()
	}
	selected := m.sessionList[m.sessionPick].SessionID
	m.clearSessionPick()
	m.phase = PhaseIdle
	if selected == m.sessionID {
		m.hint = defaultHint()
		return m.focusInputCmd()
	}
	m.applySessionSwitch(selected)
	return tea.Batch(m.hydrateHistoryCmd(), m.focusInputCmd())
}

func sessionPickerInitialPick(sessions []client.ChatSessionSummary, currentID string) int {
	for i, sess := range sessions {
		if sess.SessionID == currentID {
			return i
		}
	}
	return 0
}

func (m *Model) updateSessionsList(msg sessionsListMsg) (*Model, tea.Cmd) {
	if msg.err != "" {
		m.hint = msg.err
		return m, m.focusInputCmd()
	}
	if len(msg.sessions) == 0 {
		m.appendSystem("No sessions found")
		return m, m.focusInputCmd()
	}
	m.sessionList = msg.sessions
	m.sessionPick = sessionPickerInitialPick(msg.sessions, m.sessionID)
	m.phase = PhaseSessionPick
	m.hint = "Select a session to switch"
	return m, m.focusInputCmd()
}

func (m *Model) sessionsListCmd() tea.Cmd {
	cl := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
		defer cancel()
		sessions, _, err := cl.ChatAgent.ListSessions(ctx, "", 20)
		if err != nil {
			return sessionsListMsg{err: err.Error()}
		}
		return sessionsListMsg{sessions: sessions}
	}
}

func (m *Model) handleSlashSessions() (*Model, tea.Cmd) {
	if m.phase == PhaseStreaming || m.phase == PhaseConfirming {
		m.hint = "Finish the current action before switching sessions"
		return m, nil
	}
	return m, m.sessionsListCmd()
}

func (m *Model) handleSlashSession(cmd, args string) (*Model, tea.Cmd, bool) {
	switch cmd {
	case "new":
		return m, m.sessionNewCmd(), true
	case "end":
		return m, m.sessionEndCmd(), true
	case "status":
		m.appendSystem(SessionStatusText(m.sessionID, m.messageCount))
		return m, m.focusInputCmd(), true
	case "context":
		return m, m.contextUsageCmd(), true
	case "compact":
		return m, m.sessionCompactCmd(), true
	case "resume":
		m.transcript.Reset()
		m.streamOverlay.Reset()
		m.messageCount = 0
		return m, m.hydrateHistoryCmd(), true
	case "sessions":
		next, cmd := m.handleSlashSessions()
		return next, cmd, true
	case "export":
		return m, m.sessionExportCmd(args), true
	default:
		return m, nil, false
	}
}
