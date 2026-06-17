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
		{name: "empty prefix lists all", prefix: "", wantNames: []string{"help", "new", "end", "status", "context", "compact", "resume", "sessions", "permission", "export", "auth", "file", "clear", "quit"}},
		{name: "help prefix", prefix: "he", wantNames: []string{"help"}},
		{name: "shared prefix", prefix: "s", wantNames: []string{"status", "sessions"}},
		{name: "compact prefix", prefix: "co", wantNames: []string{"context", "compact"}},
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
		{name: "compact command", cmd: SlashCommand{name: "compact", desc: "Compress long session history"}, want: "/compact"},
		{name: "optional path argument", cmd: SlashCommand{name: "export", args: "<path>"}, want: "/export "},
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
		{name: "export without path", line: "/export", want: true},
		{name: "export placeholder path", line: "/export [path]", want: true},
		{name: "export with path", line: "/export ./out/chat", want: true},
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
		{name: "enter fills file placeholder", start: "/", pick: 11, wantValue: "/file ", wantRun: false},
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

func slashSuggestKey(code rune, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: mod}
}

func TestHandleSlashSuggestKey(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		key       tea.KeyPressMsg
		wantValue string
		wantPick  int
		wantMenu  bool
	}{
		{name: "tab completes first match", start: "/he", key: slashSuggestKey(tea.KeyTab, 0), wantValue: "/help", wantPick: 0, wantMenu: true},
		{name: "ctrl n selects next", start: "/", key: slashSuggestKey('n', tea.ModCtrl), wantValue: "/", wantPick: 1, wantMenu: true},
		{name: "ctrl p wraps selection", start: "/", key: slashSuggestKey('p', tea.ModCtrl), wantValue: "/", wantPick: len(slashCommands) - 1, wantMenu: true},
		{name: "esc dismisses menu", start: "/", key: slashSuggestKey(tea.KeyEscape, 0), wantValue: "/", wantPick: 0, wantMenu: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.input.SetValue(tt.start)
			m.syncSlashSuggest()
			handled := m.handleSlashSuggestKey(tt.key)
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
		{name: "shows navigation hint", value: "/", wantSub: "Tab complete"},
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
