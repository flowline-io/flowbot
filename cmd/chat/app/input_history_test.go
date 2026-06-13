package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func newInputHistory(entries ...string) inputHistory {
	return inputHistory{entries: append([]string(nil), entries...), index: -1}
}

func TestInputHistoryPush(t *testing.T) {
	tests := []struct {
		name      string
		push      []string
		wantCount int
		wantLast  string
	}{
		{name: "stores message", push: []string{"hello"}, wantCount: 1, wantLast: "hello"},
		{name: "stores slash command", push: []string{"/help"}, wantCount: 1, wantLast: "/help"},
		{name: "skips duplicate consecutive", push: []string{"a", "a", "b"}, wantCount: 2, wantLast: "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInputHistory()
			for _, text := range tt.push {
				h.push(text)
			}
			assert.Len(t, h.entries, tt.wantCount)
			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantLast, h.entries[len(h.entries)-1])
			}
		})
	}
}

func TestInputHistoryNavigateUp(t *testing.T) {
	tests := []struct {
		name      string
		entries   []string
		current   string
		steps     int
		wantValue string
		wantIndex int
		wantDraft string
	}{
		{name: "empty history", entries: nil, current: "draft", steps: 1, wantValue: "", wantIndex: -1, wantDraft: ""},
		{name: "first up recalls latest", entries: []string{"one", "two"}, current: "draft", steps: 1, wantValue: "two", wantIndex: 1, wantDraft: "draft"},
		{name: "second up goes older", entries: []string{"one", "two"}, current: "draft", steps: 2, wantValue: "one", wantIndex: 0, wantDraft: "draft"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInputHistory(tt.entries...)
			var value string
			var ok bool
			current := tt.current
			for i := 0; i < tt.steps; i++ {
				value, ok = h.navigateUp(current)
				if !ok {
					break
				}
				current = value
			}
			if tt.wantValue == "" {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantIndex, h.index)
			assert.Equal(t, tt.wantDraft, h.draft)
		})
	}
}

func TestInputHistoryNavigateDown(t *testing.T) {
	tests := []struct {
		name      string
		entries   []string
		startIdx  int
		draft     string
		steps     int
		wantValue string
		wantIndex int
	}{
		{name: "not browsing", entries: []string{"one"}, startIdx: -1, steps: 1, wantValue: "", wantIndex: -1},
		{name: "down to draft", entries: []string{"one", "two"}, startIdx: 1, draft: "typed", steps: 1, wantValue: "typed", wantIndex: -1},
		{name: "down to newer entry", entries: []string{"one", "two"}, startIdx: 0, draft: "typed", steps: 1, wantValue: "two", wantIndex: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInputHistory(tt.entries...)
			h.index = tt.startIdx
			h.draft = tt.draft
			var value string
			var ok bool
			for i := 0; i < tt.steps; i++ {
				value, ok = h.navigateDown()
			}
			if tt.wantValue == "" {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantIndex, h.index)
		})
	}
}

func TestHandleInputHistoryKey(t *testing.T) {
	tests := []struct {
		name      string
		entries   []string
		start     string
		keys      []rune
		wantValue string
		wantIndex int
	}{
		{name: "up recalls message", entries: []string{"hello world"}, start: "", keys: []rune{tea.KeyUp}, wantValue: "hello world", wantIndex: 0},
		{name: "up recalls command", entries: []string{"/status"}, start: "", keys: []rune{tea.KeyUp}, wantValue: "/status", wantIndex: 0},
		{name: "down restores draft", entries: []string{"hello"}, start: "draft line", keys: []rune{tea.KeyUp, tea.KeyDown}, wantValue: "draft line", wantIndex: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.inputHist.entries = append([]string(nil), tt.entries...)
			m.input.SetValue(tt.start)
			var handled bool
			for _, key := range tt.keys {
				handled = m.handleInputHistoryKey(tea.KeyPressMsg{Code: key})
				assert.True(t, handled)
			}
			assert.Equal(t, tt.wantValue, m.input.Value())
			assert.Equal(t, tt.wantIndex, m.inputHist.index)
		})
	}
}

func TestInputHistoryOverridesSlashSuggest(t *testing.T) {
	tests := []struct {
		name      string
		entries   []string
		start     string
		key       rune
		wantValue string
	}{
		{name: "up recalls over slash menu", entries: []string{"hello"}, start: "/", key: tea.KeyUp, wantValue: "hello"},
		{name: "down clears slash menu", entries: []string{"/help"}, start: "/help", key: tea.KeyDown, wantValue: ""},
		{name: "up recalls slash command", entries: []string{"/status", "hello"}, start: "/", key: tea.KeyUp, wantValue: "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.inputHist.entries = append([]string(nil), tt.entries...)
			m.input.SetValue(tt.start)
			m.syncSlashSuggest()
			if tt.name != "down clears slash menu" {
				assert.True(t, m.slashSuggestActive())
			}
			if tt.name == "down clears slash menu" {
				m.inputHist.index = 0
				m.inputHist.draft = "draft"
			}
			assert.True(t, m.handleInputHistoryKey(tea.KeyPressMsg{Code: tt.key}))
			if tt.wantValue == "" {
				assert.Equal(t, "draft", m.input.Value())
			} else {
				assert.Equal(t, tt.wantValue, m.input.Value())
			}
			assert.False(t, m.slashSuggestActive())
		})
	}
}
