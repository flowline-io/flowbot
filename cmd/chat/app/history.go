package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/flowline-io/flowbot/pkg/client"
)

const chatSessionFile = "chat_session"

// SaveSessionID persists the active session id for resume.
func SaveSessionID(profile, sessionID string) error {
	cfgDir, err := store.GetConfigDir()
	if err != nil {
		return err
	}
	name := chatSessionFile
	if profile != "" {
		name += "." + profile
	}
	path := filepath.Join(cfgDir, name)
	return os.WriteFile(path, []byte(sessionID), 0600)
}

// LoadSessionID reads a persisted session id.
func LoadSessionID(profile string) (string, error) {
	cfgDir, err := store.GetConfigDir()
	if err != nil {
		return "", err
	}
	name := chatSessionFile
	if profile != "" {
		name += "." + profile
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// ClearSessionID removes the persisted session id.
func ClearSessionID(profile string) error {
	cfgDir, err := store.GetConfigDir()
	if err != nil {
		return err
	}
	name := chatSessionFile
	if profile != "" {
		name += "." + profile
	}
	path := filepath.Join(cfgDir, name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// FormatHistoryLine renders one transcript line for the viewport.
func FormatHistoryLine(role, text string, styles Styles) string {
	switch role {
	case "user":
		return styles.UserMsg.Render("● "+text) + "\n"
	default:
		return styles.Assistant.Render(text) + "\n"
	}
}

// FormatSystemLine renders a neutral system/status line in the transcript.
func FormatSystemLine(text string, styles Styles) string {
	return styles.Hint.Render(text) + "\n"
}

// FormatSeparator returns a horizontal rule between messages.
func FormatSeparator(width int, styles Styles) string {
	if width < 20 {
		width = 40
	}
	line := repeatRune('─', width)
	return styles.Rule.Render(line)
}

func repeatRune(r rune, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = r
	}
	return string(b)
}

// SessionStatusText formats /status output.
func SessionStatusText(sessionID string, messageCount int) string {
	return fmt.Sprintf("Session: %s · messages: %d", sessionID, messageCount)
}

// EstimateHistoryTokens approximates context usage from persisted message text.
func EstimateHistoryTokens(msgs []client.ChatHistoryMessage) int {
	total := 0
	for _, msg := range msgs {
		total += estimateTokens(len(msg.Text))
	}
	return total
}

func (m *Model) applyHistoryUsage(tokens int) {
	window := m.effectiveContextWindow()
	m.status.ContextWindow = window
	m.status.TotalTokens = tokens
	m.status.ContextPercent = contextUsagePercent(tokens, window, 0)
}

func (m *Model) resetSessionUsage() {
	m.status.TotalTokens = 0
	m.status.ContextPercent = 0
}
