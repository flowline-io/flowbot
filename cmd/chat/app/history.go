package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	agentmsg "github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/client"
)

const (
	assistantMarker = "◆ "
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

// sessionCacheHint returns a status-line warning when local session persistence fails.
func sessionCacheHint(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("Warning: could not persist session locally: %v", err)
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

// resumeSessionID loads a persisted session id, validates it with the server,
// and creates a fresh session when the saved one no longer exists.
func resumeSessionID(ctx context.Context, cl *client.Client, profile string) (string, error) {
	sessionID, _ := LoadSessionID(profile)
	if sessionID != "" {
		if _, err := cl.ChatAgent.ListMessages(ctx, sessionID); err != nil {
			if client.IsNotFound(err) {
				_ = ClearSessionID(profile)
				sessionID = ""
			} else {
				return "", err
			}
		}
	}
	if sessionID == "" {
		id, err := cl.ChatAgent.CreateSession(ctx)
		if err != nil {
			return "", err
		}
		sessionID = id
		if err := SaveSessionID(profile, sessionID); err != nil {
			return "", fmt.Errorf("save session id: %w", err)
		}
	}
	return sessionID, nil
}

// FormatHistoryLine renders one transcript line for the viewport.
func FormatHistoryLine(role, text string, styles *Styles) string {
	switch role {
	case "user":
		content := styles.UserMsg.Render("● ") + styles.UserMsg.Render(text)
		return styles.UserPanel.Render(content) + "\n"
	case "assistant":
		return styles.AssistantBar.Render(assistantMarker+text) + "\n"
	default:
		return styles.Assistant.Render(text) + "\n"
	}
}

// FormatThinkingBlock renders provider reasoning above the assistant reply.
func FormatThinkingBlock(text string, width int, styles *Styles) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	marker := styles.Thinking.Render("◇ Thinking")
	body := RenderMarkdown(text, width)
	if body == "" {
		return styles.ThinkingPanel.Render(marker) + "\n"
	}
	block := marker + "\n" + styles.Thinking.Render(strings.TrimRight(body, "\n"))
	return styles.ThinkingPanel.Render(block) + "\n"
}

// FormatAssistantBlock renders agent markdown with a left marker on the first line.
func FormatAssistantBlock(text string, width int, styles *Styles) string {
	text = agentmsg.SanitizeAssistantDisplayText(text)
	if strings.TrimSpace(text) == "" {
		return ""
	}
	marker := styles.AssistantBar.Render(assistantMarker)
	body := RenderMarkdown(text, width)
	if body == "" {
		return styles.AssistantPanel.Render(marker) + "\n"
	}
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	idx := firstNonEmptyLine(lines)
	if idx < 0 {
		return styles.AssistantPanel.Render(marker) + "\n"
	}
	lines = lines[idx:]
	lines[0] = marker + lines[0]
	lines = indentAssistantContinuation(lines)
	block := strings.Join(lines, "\n")
	return styles.AssistantPanel.Render(block) + "\n"
}

func indentAssistantContinuation(lines []string) []string {
	if len(lines) <= 1 {
		return lines
	}
	out := make([]string, len(lines))
	out[0] = lines[0]
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(stripANSI(lines[i])) == "" {
			out[i] = lines[i]
			continue
		}
		out[i] = "  " + lines[i]
	}
	return out
}

func firstNonEmptyLine(lines []string) int {
	for i, line := range lines {
		if strings.TrimSpace(stripANSI(line)) != "" {
			return i
		}
	}
	return -1
}

// FormatHistoryMessages renders persisted messages with separators only between turns.
func FormatHistoryMessages(msgs []client.ChatHistoryMessage, width int, styles *Styles) string {
	msgs = coalesceHistoryAssistantMessages(msgs)
	var b strings.Builder
	for _, message := range msgs {
		if message.Role == "user" {
			if b.Len() > 0 {
				writeBuilder(&b, FormatSeparator(width, styles)+"\n")
			}
			writeBuilder(&b, FormatHistoryLine("user", message.Text, styles))
		} else if block := FormatAssistantBlock(message.Text, max(width-4, 20), styles); block != "" {
			writeBuilder(&b, block)
		}
	}
	return b.String()
}

func coalesceHistoryAssistantMessages(msgs []client.ChatHistoryMessage) []client.ChatHistoryMessage {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]client.ChatHistoryMessage, 0, len(msgs))
	var toolBatch []client.ChatHistoryMessage
	flushTools := func() {
		if len(toolBatch) == 0 {
			return
		}
		texts := make([]string, 0, len(toolBatch))
		for _, item := range toolBatch {
			texts = append(texts, item.Text)
		}
		merged := agentmsg.CoalesceAssistantHistoryMessages(texts)
		for _, text := range merged {
			item := toolBatch[len(toolBatch)-1]
			item.Text = text
			out = append(out, item)
		}
		toolBatch = nil
	}
	for _, message := range msgs {
		if message.Role == "assistant" && isHistoryToolSnapshot(message.Text) {
			toolBatch = append(toolBatch, message)
			continue
		}
		flushTools()
		out = append(out, message)
	}
	flushTools()
	return out
}

func isHistoryToolSnapshot(text string) bool {
	if agentmsg.IsToolCallPayload(text) {
		return true
	}
	return agentmsg.IsAssistantToolSummary(text)
}

// FormatSystemLine renders a neutral system/status line in the transcript.
func FormatSystemLine(text string, styles *Styles) string {
	return styles.System.Render("· "+text) + "\n"
}

// FormatSeparator returns a horizontal rule between conversation turns.
func FormatSeparator(width int, styles *Styles) string {
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
func SessionStatusText(sessionID string, messageCount int, mode string) string {
	line := fmt.Sprintf("Session: %s · messages: %d", sessionID, messageCount)
	if mode == sessionModePlan {
		line += " · mode: plan (read-only)"
	} else {
		line += " · mode: normal"
	}
	return line
}

// EstimateHistoryTokens approximates context usage from persisted message text.
func EstimateHistoryTokens(msgs []client.ChatHistoryMessage) int {
	total := 0
	for _, item := range msgs {
		total += estimateTokens(len(item.Text))
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
