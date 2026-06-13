package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const maxSlashSuggestLines = 8

// SlashCommand describes one slash command entry for autocomplete.
type SlashCommand struct {
	name string
	desc string
	args string
}

var slashCommands = []SlashCommand{
	{name: "help", desc: "Show this help"},
	{name: "new", desc: "Create a new session"},
	{name: "end", desc: "Close current session"},
	{name: "status", desc: "Show session info"},
	{name: "context", desc: "Show context usage breakdown"},
	{name: "resume", desc: "Reload saved session history"},
	{name: "auth", desc: "Show auth configuration", args: "status"},
	{name: "file", desc: "Attach local file to next message", args: "<path>"},
	{name: "clear", desc: "Clear view (keep session)"},
	{name: "quit", desc: "Exit flowbot-chat"},
}

// MatchSlashCommands returns commands whose name prefix-matches prefix.
func MatchSlashCommands(prefix string) []SlashCommand {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	var matches []SlashCommand
	for _, cmd := range slashCommands {
		if prefix == "" || strings.HasPrefix(cmd.name, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

func slashCompleteActive(line string) bool {
	if !strings.HasPrefix(line, "/") {
		return false
	}
	body := strings.TrimPrefix(line, "/")
	return !strings.Contains(body, " ")
}

func (m *Model) slashSuggestActive() bool {
	return len(m.slashMatches) > 0 && m.phase != PhaseStreaming && m.phase != PhaseConfirming
}

func (m *Model) syncSlashSuggest() {
	line := m.input.Value()
	if !slashCompleteActive(line) {
		m.clearSlashSuggest()
		return
	}
	prefix := strings.TrimPrefix(line, "/")
	matches := MatchSlashCommands(prefix)
	if len(matches) == 0 {
		m.clearSlashSuggest()
		return
	}
	if m.slashPick >= len(matches) {
		m.slashPick = 0
	}
	m.slashMatches = matches
}

func (m *Model) clearSlashSuggest() {
	m.slashMatches = nil
	m.slashPick = 0
}

func (m *Model) applySlashCompletion() {
	if len(m.slashMatches) == 0 || m.slashPick >= len(m.slashMatches) {
		return
	}
	m.input.SetValue(formatSlashCommand(m.slashMatches[m.slashPick]))
}

func formatSlashCommand(cmd SlashCommand) string {
	switch {
	case cmd.args == "":
		return "/" + cmd.name
	case strings.HasPrefix(cmd.args, "<"):
		return "/" + cmd.name + " "
	default:
		return "/" + cmd.name + " " + cmd.args
	}
}

// slashInputReadyToRun reports whether a slash line is complete enough to execute.
func slashInputReadyToRun(line string) bool {
	cmd, args, ok := ParseSlashCommand(line)
	if !ok || cmd == "" {
		return false
	}
	switch cmd {
	case "file":
		return strings.TrimSpace(args) != ""
	default:
		return true
	}
}

func (m *Model) acceptSlashSelection() (text string, run bool) {
	if !m.slashSuggestActive() {
		return "", false
	}
	m.applySlashCompletion()
	line := m.input.Value()
	m.syncSlashSuggest()
	return strings.TrimSpace(line), slashInputReadyToRun(line)
}

func (m *Model) handleSlashSuggestKey(msg tea.KeyMsg) bool {
	if !m.slashSuggestActive() {
		return false
	}
	switch msg.Key().Code {
	case tea.KeyTab:
		if msg.Key().Mod&tea.ModShift != 0 {
			m.slashPick = (m.slashPick + len(m.slashMatches) - 1) % len(m.slashMatches)
			return true
		}
		m.applySlashCompletion()
		return true
	case tea.KeyDown:
		m.slashPick = (m.slashPick + 1) % len(m.slashMatches)
		return true
	case tea.KeyUp:
		m.slashPick = (m.slashPick + len(m.slashMatches) - 1) % len(m.slashMatches)
		return true
	case tea.KeyEscape:
		m.clearSlashSuggest()
		return true
	default:
		return false
	}
}

func (m *Model) renderSlashSuggestions() string {
	if len(m.slashMatches) == 0 {
		return ""
	}
	var b strings.Builder
	limit := min(len(m.slashMatches), maxSlashSuggestLines)
	for i := range limit {
		cmd := m.slashMatches[i]
		label := fmt.Sprintf("/%-8s %s", cmd.name, cmd.desc)
		if i == m.slashPick {
			writeBuilder(&b, m.styles.InputPrompt.Render("▸ "+label))
		} else {
			writeBuilder(&b, m.styles.Hint.Render("  "+label))
		}
		writeBuilder(&b, "\n")
	}
	if len(m.slashMatches) > maxSlashSuggestLines {
		writeBuilder(&b, m.styles.Hint.Render(fmt.Sprintf("  … %d more", len(m.slashMatches)-maxSlashSuggestLines)))
		writeBuilder(&b, "\n")
	}
	writeBuilder(&b, m.styles.Hint.Render("  Tab/Enter accept · ↑↓ select · Esc dismiss"))
	writeBuilder(&b, "\n")
	return b.String()
}

func (m *Model) slashSuggestHeight() int {
	if len(m.slashMatches) == 0 {
		return 0
	}
	lines := len(m.slashMatches)
	if lines > maxSlashSuggestLines {
		lines = maxSlashSuggestLines + 1
	}
	return lines + 1
}

func (m *Model) updateInput(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyMsg); ok {
		if m.handleSlashSuggestKey(key) {
			return nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.syncSlashSuggest()
	return cmd
}
