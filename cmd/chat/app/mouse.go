package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// transcriptRegionBounds returns the inclusive top row and exclusive bottom row
// of the scrollable conversation viewport in terminal coordinates.
func (m *Model) transcriptRegionBounds() (top, bottom int) {
	if m.width <= 0 || m.height <= 0 {
		return 0, 0
	}
	top = lipgloss.Height(m.renderTopSection())
	bottom = top + m.viewport.Height()
	return top, bottom
}

// isInTranscriptRegion reports whether a terminal row belongs to the transcript viewport.
func (m *Model) isInTranscriptRegion(y int) bool {
	top, bottom := m.transcriptRegionBounds()
	return y >= top && y < bottom
}

// handleMouseWheel routes wheel events to the transcript viewport when the
// cursor is over the conversation area.
func (m *Model) handleMouseWheel(msg tea.MouseWheelMsg) (*Model, tea.Cmd) {
	if !m.isInTranscriptRegion(msg.Y) {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
