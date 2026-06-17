package app

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	confirmChoiceApprove = 0
	confirmChoiceDeny    = 1
)

var confirmChoices = []string{"Approve", "Deny"}

func (m *Model) renderConfirmPrompt() string {
	if m.phase != PhaseConfirming {
		return ""
	}
	var body strings.Builder
	writeBuilder(&body, "Confirm tool "+m.confirmTool)
	writeBuilder(&body, "\n")
	writeBuilder(&body, m.confirmSummary)
	writeBuilder(&body, "\n")
	for i, label := range confirmChoices {
		line := "  " + label
		if i == m.confirmPick {
			writeBuilder(&body, m.styles.InputPrompt.Render("▸ "+label))
		} else {
			writeBuilder(&body, m.styles.Hint.Render(line))
		}
		writeBuilder(&body, "\n")
	}
	writeBuilder(&body, m.styles.Hint.Render("  ↑↓ select · Enter confirm · Esc deny · Ctrl+C cancel"))
	return m.styles.ConfirmBox.Render(body.String())
}

func (m *Model) confirmFooterHeight() int {
	if m.phase != PhaseConfirming {
		return 0
	}
	lines := 3 + len(confirmChoices)
	return lines + 2
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.phase != PhaseConfirming {
		return false, nil
	}
	switch msg.Key().Code {
	case tea.KeyUp:
		m.confirmPick = (m.confirmPick + len(confirmChoices) - 1) % len(confirmChoices)
		return true, nil
	case tea.KeyDown:
		m.confirmPick = (m.confirmPick + 1) % len(confirmChoices)
		return true, nil
	case tea.KeyEnter, tea.KeyTab:
		return true, m.submitConfirmChoice(m.confirmPick == confirmChoiceApprove)
	case tea.KeyEscape:
		return true, m.submitConfirmChoice(false)
	}
	switch msg.String() {
	case "y", "Y":
		return true, m.submitConfirmChoice(true)
	case "n", "N":
		return true, m.submitConfirmChoice(false)
	}
	return false, nil
}

func (m *Model) submitConfirmChoice(approved bool) tea.Cmd {
	id := m.pendingConfirmID
	m.clearConfirm()
	m.phase = PhaseStreaming
	m.hint = runControlHint(
		m.client.ChatAgent.Confirm(context.Background(), m.sessionID, id, approved),
		"Ctrl+C cancel run",
		"Confirm failed — server may still be waiting",
	)
	return m.focusInputCmd()
}

func confirmChoiceLabel(pick int) string {
	if pick < 0 || pick >= len(confirmChoices) {
		return ""
	}
	return confirmChoices[pick]
}
