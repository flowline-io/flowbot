package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestTranscriptContentPos(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		termX    int
		wantLine int
		wantCol  int
		wantOK   bool
	}{
		{
			name:     "maps first visible line",
			content:  "hello world",
			termX:    2,
			wantLine: 0,
			wantCol:  2,
			wantOK:   true,
		},
		{
			name:     "clamps column past line width",
			content:  "hi",
			termX:    40,
			wantLine: 0,
			wantCol:  2,
			wantOK:   true,
		},
		{
			name:    "empty transcript",
			content: "",
			termX:   0,
			wantOK:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 40
			if tt.content != "" {
				writeBuilder(&m.transcript, tt.content)
			}
			m.syncViewport()
			top, _ := m.transcriptRegionBounds()
			line, col, ok := m.transcriptContentPos(tt.termX, top)
			assert.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				return
			}
			assert.Equal(t, tt.wantLine, line)
			assert.Equal(t, tt.wantCol, col)
		})
	}
}

func TestSelectedPlainText(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model)
		want  string
	}{
		{
			name: "single line selection",
			setup: func(m *Model) {
				writeBuilder(&m.transcript, "hello world\n")
				m.selActive = true
				m.selAnchor = textPos{line: 0, col: 0}
				m.selFocus = textPos{line: 0, col: 5}
			},
			want: "hello",
		},
		{
			name: "multi line selection",
			setup: func(m *Model) {
				writeBuilder(&m.transcript, "line one\nline two\n")
				m.selActive = true
				m.selAnchor = textPos{line: 0, col: 5}
				m.selFocus = textPos{line: 1, col: 4}
			},
			want: "one\nline",
		},
		{
			name: "whitespace only returns empty",
			setup: func(m *Model) {
				writeBuilder(&m.transcript, "   \n")
				m.selActive = true
				m.selAnchor = textPos{line: 0, col: 0}
				m.selFocus = textPos{line: 0, col: 3}
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			tt.setup(m)
			assert.Equal(t, tt.want, m.selectedPlainText())
		})
	}
}

func TestHandleMouseReleaseCopiesSelection(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(m *Model) int
		releaseX     int
		wantHint     string
		wantCmd      bool
		wantSelected bool
	}{
		{
			name: "release after drag copies text",
			setup: func(m *Model) int {
				writeBuilder(&m.transcript, FormatHistoryLine("user", "copy me", m.styles))
				m.syncViewport()
				top, _ := m.transcriptRegionBounds()
				m.selActive = true
				m.selDragging = true
				m.selAnchor = textPos{line: 0, col: 0}
				m.selFocus = textPos{line: 0, col: 8}
				return top
			},
			releaseX:     8,
			wantHint:     copiedHint,
			wantCmd:      true,
			wantSelected: false,
		},
		{
			name: "release without drag does nothing",
			setup: func(m *Model) int {
				writeBuilder(&m.transcript, "hello\n")
				m.syncViewport()
				top, _ := m.transcriptRegionBounds()
				return top
			},
			releaseX:     2,
			wantHint:     defaultHint(),
			wantCmd:      false,
			wantSelected: false,
		},
		{
			name: "empty selection does not copy",
			setup: func(m *Model) int {
				writeBuilder(&m.transcript, "hello\n")
				m.syncViewport()
				top, _ := m.transcriptRegionBounds()
				m.selActive = true
				m.selDragging = true
				m.selAnchor = textPos{line: 0, col: 2}
				m.selFocus = textPos{line: 0, col: 2}
				return top
			},
			releaseX:     2,
			wantHint:     defaultHint(),
			wantCmd:      false,
			wantSelected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 40
			top := tt.setup(m)
			updated, cmd := m.handleMouseRelease(tea.MouseReleaseMsg{
				X:      tt.releaseX,
				Y:      top,
				Button: tea.MouseLeft,
			})
			assert.Equal(t, tt.wantHint, updated.hint)
			assert.Equal(t, tt.wantSelected, updated.selActive)
			if tt.wantCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}
		})
	}
}

func TestHandleMouseClickClearsSelectionOutsideTranscript(t *testing.T) {
	tests := []struct {
		name string
		y    int
	}{
		{name: "header click", y: 0},
		{name: "footer click", y: 39},
		{name: "status row click", y: 38},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 40
			writeBuilder(&m.transcript, "hello\n")
			m.syncViewport()
			m.selActive = true
			m.selDragging = true
			updated, cmd := m.handleMouseClick(tea.MouseClickMsg{
				X:      1,
				Y:      tt.y,
				Button: tea.MouseLeft,
			})
			assert.Nil(t, cmd)
			assert.False(t, updated.selActive)
		})
	}
}

func TestRenderTranscriptSelectionHighlight(t *testing.T) {
	tests := []struct {
		name               string
		setup              func(m *Model)
		wantBG             bool
		wantSameAsViewport bool
	}{
		{
			name: "highlights selected span",
			setup: func(m *Model) {
				writeBuilder(&m.transcript, "hello world\n")
				m.selActive = true
				m.selAnchor = textPos{line: 0, col: 0}
				m.selFocus = textPos{line: 0, col: 5}
			},
			wantBG: true,
		},
		{
			name: "highlights styled user line",
			setup: func(m *Model) {
				writeBuilder(&m.transcript, FormatHistoryLine("user", "hello", m.styles))
				m.selActive = true
				m.selAnchor = textPos{line: 0, col: 0}
				m.selFocus = textPos{line: 0, col: 4}
			},
			wantBG: true,
		},
		{
			name: "inactive selection uses viewport view",
			setup: func(m *Model) {
				writeBuilder(&m.transcript, "hello\n")
			},
			wantSameAsViewport: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 40
			tt.setup(m)
			m.syncViewport()
			rendered := m.renderTranscript()
			if tt.wantSameAsViewport {
				assert.Equal(t, m.viewport.View(), rendered)
				return
			}
			if tt.wantBG {
				assert.Contains(t, rendered, "48;2;")
			}
		})
	}
}
