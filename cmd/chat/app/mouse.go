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

// handleMouse routes mouse events for transcript scrolling and text selection.
func (m *Model) handleMouse(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)
	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)
	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg)
	case tea.MouseReleaseMsg:
		return m.handleMouseRelease(msg)
	default:
		return m, nil
	}
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

func (m *Model) handleMouseClick(msg tea.MouseClickMsg) (*Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	if !m.isInTranscriptRegion(msg.Y) {
		m.clearSelection()
		return m, nil
	}
	line, col, ok := m.transcriptContentPos(msg.X, msg.Y)
	if !ok {
		m.clearSelection()
		return m, nil
	}
	m.selActive = true
	m.selDragging = true
	m.selAnchor = textPos{line: line, col: col}
	m.selFocus = textPos{line: line, col: col}
	return m, nil
}

func (m *Model) handleMouseMotion(msg tea.MouseMotionMsg) (*Model, tea.Cmd) {
	if !m.selDragging || msg.Button != tea.MouseLeft {
		return m, nil
	}
	line, col, ok := m.transcriptContentPos(msg.X, msg.Y)
	if !ok {
		return m, nil
	}
	m.selFocus = textPos{line: line, col: col}
	return m, nil
}

func (m *Model) handleMouseRelease(msg tea.MouseReleaseMsg) (*Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft || !m.selDragging {
		return m, nil
	}
	m.selDragging = false
	if m.isInTranscriptRegion(msg.Y) {
		if line, col, ok := m.transcriptContentPos(msg.X, msg.Y); ok {
			m.selFocus = textPos{line: line, col: col}
		}
	}
	text := m.selectedPlainText()
	if text == "" {
		m.clearSelection()
		return m, nil
	}
	m.hint = copiedHint
	m.clearSelection()
	return m, copyToClipboardCmd(text)
}
