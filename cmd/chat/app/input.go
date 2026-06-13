package app

import (
	tea "charm.land/bubbletea/v2"
)

const inputPromptText = " ❯ "

func (m *Model) focusInputCmd() tea.Cmd {
	return m.input.Focus()
}
