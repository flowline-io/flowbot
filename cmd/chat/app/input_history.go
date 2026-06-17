package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

const maxInputHistoryEntries = 500

// inputHistory stores submitted messages and slash commands for up/down recall in memory.
type inputHistory struct {
	entries []string
	index   int // -1 when not browsing history
	draft   string
}

func (h *inputHistory) push(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if n := len(h.entries); n > 0 && h.entries[n-1] == text {
		h.resetBrowse()
		return
	}
	h.entries = append(h.entries, text)
	if len(h.entries) > maxInputHistoryEntries {
		h.entries = h.entries[len(h.entries)-maxInputHistoryEntries:]
	}
	h.resetBrowse()
}

func (h *inputHistory) resetBrowse() {
	h.index = -1
	h.draft = ""
}

func (h *inputHistory) exitBrowse() {
	h.index = -1
	h.draft = ""
}

func (h *inputHistory) navigateUp(current string) (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}
	if h.index == -1 {
		h.draft = current
		h.index = len(h.entries) - 1
		return h.entries[h.index], true
	}
	if h.index == 0 {
		return h.entries[0], true
	}
	h.index--
	return h.entries[h.index], true
}

func (h *inputHistory) navigateDown() (string, bool) {
	if len(h.entries) == 0 || h.index == -1 {
		return "", false
	}
	if h.index >= len(h.entries)-1 {
		h.index = -1
		return h.draft, true
	}
	h.index++
	return h.entries[h.index], true
}

func (m *Model) handleInputHistoryKey(msg tea.KeyMsg) bool {
	if m.phase == PhaseStreaming || m.phase == PhaseConfirming || m.phase == PhaseSessionPick {
		return false
	}
	switch msg.Key().Code {
	case tea.KeyUp:
		value, ok := m.inputHist.navigateUp(m.input.Value())
		if !ok {
			return false
		}
		m.input.SetValue(value)
		m.clearSlashSuggest()
		return true
	case tea.KeyDown:
		value, ok := m.inputHist.navigateDown()
		if !ok {
			return false
		}
		m.input.SetValue(value)
		m.clearSlashSuggest()
		return true
	default:
		if m.inputHist.index != -1 && inputHistoryEditKey(msg) {
			m.inputHist.exitBrowse()
		}
		return false
	}
}

func inputHistoryEditKey(msg tea.KeyMsg) bool {
	switch msg.Key().Code {
	case tea.KeyUp, tea.KeyDown, tea.KeyEnter, tea.KeyTab, tea.KeyEscape:
		return false
	default:
		return true
	}
}
