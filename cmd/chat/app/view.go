package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// View renders the chat TUI.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

func (m *Model) render() string {
	if m.width == 0 {
		return "Loading...\n"
	}
	if m.errMsg != "" {
		return m.styles.Warning.Render("Error: "+m.errMsg) + "\n"
	}
	if m.height <= 0 {
		return "Loading...\n"
	}
	if m.resourceOverlay != nil {
		return m.renderResourceOverlay()
	}

	header := m.renderTopSection()
	footer := m.renderFooter()
	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	footerY := max(m.height-footerH, headerH)

	canvas := lipgloss.NewCanvas(m.width, m.height)
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(header).Y(0),
		lipgloss.NewLayer(m.renderTranscript()).Y(headerH),
		lipgloss.NewLayer(footer).Y(footerY),
	)
	return canvas.Compose(comp).Render()
}
