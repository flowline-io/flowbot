package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestMatchSlashCommands(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		wantNames []string
	}{
		{name: "empty prefix lists all", prefix: "", wantNames: []string{"help", "new", "end", "status", "resume", "auth", "file", "clear", "quit"}},
		{name: "help prefix", prefix: "he", wantNames: []string{"help"}},
		{name: "shared prefix", prefix: "s", wantNames: []string{"status"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchSlashCommands(tt.prefix)
			names := make([]string, len(got))
			for i, cmd := range got {
				names[i] = cmd.name
			}
			assert.Equal(t, tt.wantNames, names)
		})
	}
}

func TestSlashCompleteActive(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "slash only", line: "/", want: true},
		{name: "partial command", line: "/he", want: true},
		{name: "command with args", line: "/file ./main.go", want: false},
		{name: "plain text", line: "hello", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, slashCompleteActive(tt.line))
		})
	}
}

func TestFormatSlashCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  SlashCommand
		want string
	}{
		{name: "simple command", cmd: SlashCommand{name: "help", desc: "Show this help"}, want: "/help"},
		{name: "path argument", cmd: SlashCommand{name: "file", args: "<path>"}, want: "/file "},
		{name: "literal argument", cmd: SlashCommand{name: "auth", args: "status"}, want: "/auth status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatSlashCommand(tt.cmd))
		})
	}
}

func TestSyncSlashSuggest(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantActive bool
		wantCount  int
	}{
		{name: "shows menu for slash", value: "/", wantActive: true, wantCount: len(slashCommands)},
		{name: "filters prefix", value: "/he", wantActive: true, wantCount: 1},
		{name: "hides after args", value: "/file x", wantActive: false, wantCount: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.input.SetValue(tt.value)
			m.syncSlashSuggest()
			assert.Equal(t, tt.wantActive, m.slashSuggestActive())
			assert.Len(t, m.slashMatches, tt.wantCount)
		})
	}
}

func TestSlashInputReadyToRun(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "help command", line: "/help", want: true},
		{name: "file without path", line: "/file ", want: false},
		{name: "file with path", line: "/file ./main.go", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, slashInputReadyToRun(tt.line))
		})
	}
}

func TestAcceptSlashSelection(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		pick      int
		wantValue string
		wantRun   bool
	}{
		{name: "enter fills selected command", start: "/", pick: 1, wantValue: "/new", wantRun: true},
		{name: "enter fills file placeholder", start: "/", pick: 6, wantValue: "/file ", wantRun: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.input.SetValue(tt.start)
			m.syncSlashSuggest()
			m.slashPick = tt.pick
			text, run := m.acceptSlashSelection()
			assert.Equal(t, tt.wantValue, m.input.Value())
			assert.Equal(t, tt.wantRun, run)
			if run {
				assert.Equal(t, strings.TrimSpace(tt.wantValue), text)
			}
		})
	}
}

func TestHandleSlashSuggestKey(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		key       rune
		wantValue string
		wantPick  int
		wantMenu  bool
	}{
		{name: "tab completes first match", start: "/he", key: tea.KeyTab, wantValue: "/help", wantPick: 0, wantMenu: true},
		{name: "down selects next", start: "/", key: tea.KeyDown, wantValue: "/", wantPick: 1, wantMenu: true},
		{name: "up wraps selection", start: "/", key: tea.KeyUp, wantValue: "/", wantPick: len(slashCommands) - 1, wantMenu: true},
		{name: "esc dismisses menu", start: "/", key: tea.KeyEscape, wantValue: "/", wantPick: 0, wantMenu: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.input.SetValue(tt.start)
			m.syncSlashSuggest()
			handled := m.handleSlashSuggestKey(tea.KeyPressMsg{Code: tt.key})
			assert.True(t, handled)
			assert.Equal(t, tt.wantValue, m.input.Value())
			assert.Equal(t, tt.wantPick, m.slashPick)
			assert.Equal(t, tt.wantMenu, m.slashSuggestActive())
		})
	}
}

func TestRenderSlashSuggestions(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantSub string
	}{
		{name: "shows help entry", value: "/he", wantSub: "/help"},
		{name: "shows navigation hint", value: "/", wantSub: "Tab/Enter accept"},
		{name: "hidden without slash", value: "hello", wantSub: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.input.SetValue(tt.value)
			m.syncSlashSuggest()
			got := m.renderSlashSuggestions()
			if tt.wantSub == "" {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, tt.wantSub)
		})
	}
}
