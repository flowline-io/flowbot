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
	if summary.Mode == sessionModePlan {
		label += " · plan"
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
	if m.phase != PhaseSessionPick || len(m.picker.list) == 0 {
		return ""
	}
	var body strings.Builder
	writeBuilder(&body, "Sessions")
	writeBuilder(&body, "\n")
	limit := min(len(m.picker.list), maxSessionPickerLines)
	for i := range limit {
		writeBuilder(&body, FormatSessionRow(m.picker.list[i], m.sessionID, i == m.picker.pick, &m.styles))
		writeBuilder(&body, "\n")
	}
	if len(m.picker.list) > maxSessionPickerLines {
		writeBuilder(&body, m.styles.Hint.Render(fmt.Sprintf("  … and %d more", len(m.picker.list)-maxSessionPickerLines)))
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
	visible := len(m.picker.list)
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
		if len(m.picker.list) == 0 {
			return true, nil
		}
		m.picker.pick = (m.picker.pick + len(m.picker.list) - 1) % len(m.picker.list)
		return true, nil
	case tea.KeyDown:
		if len(m.picker.list) == 0 {
			return true, nil
		}
		m.picker.pick = (m.picker.pick + 1) % len(m.picker.list)
		return true, nil
	case tea.KeyEnter, tea.KeyTab:
		return true, m.submitSessionPick()
	case tea.KeyEscape:
		m.clearSessionPick()
		m.phase = PhaseIdle
		m.resetInputHint()
		return true, m.focusInputCmd()
	}
	return false, nil
}

func (m *Model) applySessionSwitch(id, mode string) {
	m.sessionID = id
	m.transcript.Reset()
	m.stream.overlay.Reset()
	m.messageCount = 0
	m.resetSessionUsage()
	m.splashVisible = false
	if mode != "" {
		m.finalizeSessionMode(mode)
		return
	}
	m.applySessionModeDisplay("normal")
	if hint := sessionCacheHint(SaveSessionID(m.profile, id)); hint != "" {
		m.hint = hint
	} else {
		m.hint = "Switched session"
	}
	m.syncLayout()
	m.syncViewport()
}

func (m *Model) submitSessionPick() tea.Cmd {
	if len(m.picker.list) == 0 || m.picker.pick < 0 || m.picker.pick >= len(m.picker.list) {
		m.clearSessionPick()
		m.phase = PhaseIdle
		return m.focusInputCmd()
	}
	selected := m.picker.list[m.picker.pick]
	m.clearSessionPick()
	m.phase = PhaseIdle
	if selected.SessionID == m.sessionID {
		m.resetInputHint()
		return m.focusInputCmd()
	}
	m.applySessionSwitch(selected.SessionID, selected.Mode)
	return tea.Batch(m.loadSessionModeCmd(selected.SessionID), m.hydrateHistoryCmd(), m.focusInputCmd())
}

func (m *Model) loadSessionModeCmd(sessionID string) tea.Cmd {
	cl := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
		defer cancel()
		mode, err := cl.ChatAgent.GetSessionMode(ctx, sessionID)
		if err != nil {
			return sessionModeLoadMsg{sessionID: sessionID, err: err.Error()}
		}
		return sessionModeLoadMsg{sessionID: sessionID, mode: mode}
	}
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
	m.picker.list = msg.sessions
	m.picker.pick = sessionPickerInitialPick(msg.sessions, m.sessionID)
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
		m.appendSystem(SessionStatusText(m.sessionID, m.messageCount, m.mode))
		return m, m.focusInputCmd(), true
	case "context":
		return m, m.contextUsageCmd(), true
	case "compact":
		return m, m.sessionCompactCmd(), true
	case "resume":
		m.transcript.Reset()
		m.stream.overlay.Reset()
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
