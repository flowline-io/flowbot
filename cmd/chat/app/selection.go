package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"
)

const copiedHint = "Copied to clipboard"

// textPos is a line/column position in the transcript content.
type textPos struct {
	line int
	col  int
}

// transcriptContentPos maps terminal coordinates to a transcript line and column.
func (m *Model) transcriptContentPos(termX, termY int) (line, col int, ok bool) {
	top, _ := m.transcriptRegionBounds()
	viewRow := termY - top
	if viewRow < 0 || viewRow >= m.viewport.Height() {
		return 0, 0, false
	}

	content := m.transcript.String()
	if content == "" {
		return 0, 0, false
	}

	lines := strings.Split(content, "\n")
	line = m.viewport.YOffset() + viewRow
	col = termX + m.viewport.XOffset()
	if col < 0 {
		col = 0
	}

	if line >= len(lines) {
		line = len(lines) - 1
	}
	maxCol := ansi.StringWidth(lines[line])
	if col > maxCol {
		col = maxCol
	}
	return line, col, true
}

func (m *Model) normalizedSelection() (start, end textPos) {
	start, end = m.selAnchor, m.selFocus
	if start.line > end.line || (start.line == end.line && start.col > end.col) {
		start, end = end, start
	}
	return start, end
}

func lineColToByteOffset(lines []string, line, col int) int {
	offset := 0
	for i := 0; i < line && i < len(lines); i++ {
		offset += len(lines[i]) + 1
	}
	if line >= len(lines) {
		return offset
	}
	return offset + len(ansi.Cut(lines[line], 0, col))
}

func (m *Model) selectionByteRange() (start, end int) {
	lines := strings.Split(m.transcript.String(), "\n")
	a, b := m.normalizedSelection()
	return lineColToByteOffset(lines, a.line, a.col), lineColToByteOffset(lines, b.line, b.col)
}

// selectedPlainText returns the ANSI-stripped transcript slice for the current selection.
func (m *Model) selectedPlainText() string {
	if !m.selActive {
		return ""
	}
	content := m.transcript.String()
	start, end := m.selectionByteRange()
	if start > end {
		start, end = end, start
	}
	if start == end || start >= len(content) {
		return ""
	}
	if end > len(content) {
		end = len(content)
	}
	selected := ansi.Strip(content[start:end])
	if strings.TrimSpace(selected) == "" {
		return ""
	}
	return selected
}

func (m *Model) clearSelection() {
	m.selActive = false
	m.selDragging = false
}

func selectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#5151A8")).
		Foreground(lipgloss.Color("#FFFFFF"))
}

// selectionRangesForVisibleLine returns highlight ranges for one visible transcript row.
func (m *Model) selectionRangesForVisibleLine(contentLineIdx int, displayLine string, xOff int) []lipgloss.Range {
	if !m.selActive {
		return nil
	}
	start, end := m.normalizedSelection()
	if contentLineIdx < start.line || contentLineIdx > end.line {
		return nil
	}

	colStart := 0
	colEnd := ansi.StringWidth(displayLine) + xOff
	switch {
	case contentLineIdx == start.line && contentLineIdx == end.line:
		colStart, colEnd = start.col, end.col
	case contentLineIdx == start.line:
		colStart = start.col
	case contentLineIdx == end.line:
		colEnd = end.col
	}

	colStart -= xOff
	colEnd -= xOff
	lineWidth := ansi.StringWidth(displayLine)
	if colEnd <= 0 || colStart >= lineWidth {
		return nil
	}
	colStart = max(0, colStart)
	colEnd = min(lineWidth, colEnd)
	if colStart >= colEnd {
		return nil
	}
	style := selectionStyle()
	return []lipgloss.Range{lipgloss.NewRange(colStart, colEnd, style)}
}

// renderTranscript draws the scrollable conversation area with live selection highlights.
func (m *Model) renderTranscript() string {
	if !m.selActive {
		return m.viewport.View()
	}

	w, h := m.viewport.Width(), m.viewport.Height()
	if w <= 0 || h <= 0 {
		return ""
	}

	content := m.transcript.String()
	if content == "" {
		return m.viewport.View()
	}

	allLines := strings.Split(content, "\n")
	yOff := m.viewport.YOffset()
	xOff := m.viewport.XOffset()
	visible := make([]string, 0, h)

	for viewRow := range h {
		lineIdx := yOff + viewRow
		if lineIdx >= len(allLines) {
			break
		}
		line := allLines[lineIdx]
		if xOff > 0 || ansi.StringWidth(line) > w {
			line = ansi.Cut(line, xOff, xOff+w)
		}
		if ranges := m.selectionRangesForVisibleLine(lineIdx, line, xOff); len(ranges) > 0 {
			line = lipgloss.StyleRanges(line, ranges...)
		}
		visible = append(visible, line)
	}
	for len(visible) < h {
		visible = append(visible, "")
	}
	return lipgloss.NewStyle().Width(w).Height(h).Render(strings.Join(visible, "\n"))
}

// copyToClipboardCmd writes text to the system clipboard via OSC52 and native APIs.
func copyToClipboardCmd(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	return tea.Batch(
		tea.SetClipboard(text),
		func() tea.Msg {
			_ = clipboard.WriteAll(text)
			return nil
		},
	)
}
