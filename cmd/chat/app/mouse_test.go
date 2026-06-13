package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestIsInTranscriptRegion(t *testing.T) {
	tests := []struct {
		name   string
		height int
		y      int
		want   bool
	}{
		{name: "inside transcript", height: 40, y: 12, want: true},
		{name: "header row", height: 40, y: 0, want: false},
		{name: "footer row", height: 40, y: 39, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = tt.height
			m.syncLayout()
			assert.Equal(t, tt.want, m.isInTranscriptRegion(tt.y))
		})
	}
}

func TestHandleMouseWheelScrollsTranscript(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, m *Model)
	}{
		{
			name: "wheel up from bottom scrolls earlier",
			run: func(t *testing.T, m *Model) {
				before := m.viewport.YOffset()
				top, _ := m.transcriptRegionBounds()
				updated, _ := m.handleMouseWheel(tea.MouseWheelMsg{Y: top + 1, Button: tea.MouseWheelUp})
				assert.Less(t, updated.viewport.YOffset(), before)
			},
		},
		{
			name: "wheel down after scrolling up moves toward bottom",
			run: func(t *testing.T, m *Model) {
				top, _ := m.transcriptRegionBounds()
				scrolled, _ := m.handleMouseWheel(tea.MouseWheelMsg{Y: top + 1, Button: tea.MouseWheelUp})
				before := scrolled.viewport.YOffset()
				updated, _ := scrolled.handleMouseWheel(tea.MouseWheelMsg{Y: top + 1, Button: tea.MouseWheelDown})
				assert.Greater(t, updated.viewport.YOffset(), before)
			},
		},
		{
			name: "wheel in header does not scroll",
			run: func(t *testing.T, m *Model) {
				before := m.viewport.YOffset()
				updated, _ := m.handleMouseWheel(tea.MouseWheelMsg{Y: 0, Button: tea.MouseWheelUp})
				assert.Equal(t, before, updated.viewport.YOffset())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 40
			for i := range 80 {
				m.appendSystem(fmt.Sprintf("line-%d", i))
			}
			m.syncLayout()
			tt.run(t, m)
		})
	}
}

func TestHandleMouseWheelIgnoresFooter(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		y       int
	}{
		{name: "footer wheel does not recall history", entries: []string{"hello"}, y: 39},
		{name: "footer wheel leaves slash input", entries: []string{"/help"}, y: 38},
		{name: "footer wheel keeps draft", entries: []string{"draft"}, y: 37},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 40
			m.inputHist.entries = append([]string(nil), tt.entries...)
			m.input.SetValue("typed")
			m.syncLayout()
			_, bottom := m.transcriptRegionBounds()
			y := max(bottom, tt.y)
			updated, _ := m.handleMouseWheel(tea.MouseWheelMsg{Y: y, Button: tea.MouseWheelUp})
			assert.Equal(t, "typed", updated.input.Value())
			assert.Equal(t, -1, updated.inputHist.index)
		})
	}
}

func TestViewEnablesMouseCellMotion(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{name: "standard layout", width: 80, height: 40},
		{name: "narrow terminal", width: 60, height: 24},
		{name: "tall terminal", width: 120, height: 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = tt.width
			m.height = tt.height
			view := m.View()
			assert.Equal(t, tea.MouseModeCellMotion, view.MouseMode)
		})
	}
}

func TestLongTranscriptContentIsScrollable(t *testing.T) {
	tests := []struct {
		name  string
		lines int
	}{
		{name: "many lines", lines: 80},
		{name: "extra lines", lines: 120},
		{name: "minimal overflow", lines: 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.width = 80
			m.height = 40
			var b strings.Builder
			for i := range tt.lines {
				fmt.Fprintf(&b, "line-%d\n", i)
			}
			writeBuilder(&m.transcript, b.String())
			m.syncViewport()
			assert.Positive(t, m.viewport.YOffset())
		})
	}
}
