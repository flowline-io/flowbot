package partials

import (
	"fmt"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	// ChatAgentPendingPromptKeyPrefix stores the first user prompt before navigation.
	// The create flow writes both ?prompt= and sessionStorage; the detail page must
	// always clear the storage key after consuming either source, or revisiting the
	// session from the list will re-send the first prompt.
	ChatAgentPendingPromptKeyPrefix = "flowbot-chatagent-pending:"
)

// ChatAgentDetailURL builds a session detail URL from a template containing "{id}".
func ChatAgentDetailURL(template, sessionID string) string {
	return strings.ReplaceAll(template, "{id}", sessionID)
}

// ChatAgentSessionTitle returns the display title for a session list row.
func ChatAgentSessionTitle(item model.AgentSession) string {
	if title := strings.TrimSpace(item.Title); title != "" {
		return title
	}
	if item.Flag != "" {
		return item.Flag
	}
	return "Untitled session"
}

// FormatChatAgentRelativeTime returns a compact relative timestamp (e.g. "2h", "6d").
func FormatChatAgentRelativeTime(t time.Time) string {
	return chatAgentRelativeTimeSince(t, time.Now())
}

// ChatAgentDurationLabel formats a millisecond duration for chat UI labels.
func ChatAgentDurationLabel(ms int64) string {
	if ms <= 0 {
		return ""
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

// FormatChatAgentDuration formats milliseconds for chat timing labels.
func FormatChatAgentDuration(ms int64) string {
	if ms <= 0 {
		return ""
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func chatAgentRelativeTimeSince(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	elapsed := max(now.Sub(t), 0)
	switch {
	case elapsed < time.Minute:
		return "now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh", int(elapsed.Hours()))
	case elapsed < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(elapsed.Hours()/24))
	case elapsed < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(elapsed.Hours()/(24*7)))
	default:
		return t.Format("Jan 2")
	}
}

// chatAgentSessionThumbClass returns thumbnail frame classes for a session state.
func chatAgentSessionThumbClass(state string) string {
	base := "agents-session-thumb"
	if state == "Active" {
		return base + " agents-session-thumb-active"
	}
	return base
}

// chatAgentSessionBadgeClass returns pill badge classes for the session thumbnail.
func chatAgentSessionBadgeClass(state string) string {
	switch state {
	case "Active":
		return "agents-session-badge agents-session-badge-active"
	case "Closed":
		return "agents-session-badge agents-session-badge-closed"
	default:
		return "agents-session-badge agents-session-badge-unknown"
	}
}

// ChatAgentPendingPromptKey returns the sessionStorage key for a pending first prompt.
func ChatAgentPendingPromptKey(sessionID string) string {
	return ChatAgentPendingPromptKeyPrefix + sessionID
}

// ClassifyHistoryMessage splits one persisted history row into UI-friendly chat bubbles.
func ClassifyHistoryMessage(role, text string, createdAt time.Time) []model.AgentChatMessage {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if role == "user" {
		return []model.AgentChatMessage{{
			Role:      "user",
			Kind:      "user",
			Text:      text,
			HTML:      FormatChatAgentMessageHTML("user", text),
			CreatedAt: createdAt,
		}}
	}

	if msg.IsAssistantToolSummary(text) || msg.IsToolCallPayload(text) {
		return []model.AgentChatMessage{toolMessageFromSummary(text, createdAt)}
	}

	clean, tools := stripToolPayloads(text)
	out := make([]model.AgentChatMessage, 0, len(tools)+1)
	for _, tool := range tools {
		tool.CreatedAt = createdAt
		out = append(out, tool)
	}
	clean = strings.TrimSpace(msg.SanitizeAssistantDisplayText(clean))
	clean = stripRunningToolStatus(clean)
	if clean != "" {
		out = append(out, model.AgentChatMessage{
			Role:      "assistant",
			Kind:      "assistant",
			Text:      clean,
			HTML:      FormatChatAgentMessageHTML("assistant", clean),
			CreatedAt: createdAt,
		})
	}
	return out
}

func toolMessageFromSummary(text string, createdAt time.Time) model.AgentChatMessage {
	name, args, _ := ParseToolSummaryLine(text)
	if name == "" {
		name = firstToolNameFromPayload(text)
	}
	stdout := args
	summary := text
	if msg.IsToolCallPayload(text) {
		summary = msg.SummarizeToolCallPayload(text)
		if name == "" {
			name = firstToolNameFromPayload(summary)
		}
		stdout = ""
	}
	if name == "" {
		name = "tool"
	}
	return model.AgentChatMessage{
		Role:       "assistant",
		Kind:       "tool",
		Text:       summary,
		ToolName:   name,
		ToolStatus: "completed",
		ToolStdout: stdout,
		CreatedAt:  createdAt,
	}
}

// ParseToolSummaryLine parses compact summaries like run_terminal({...}).
func ParseToolSummaryLine(text string) (name, args string, ok bool) {
	text = strings.TrimSpace(text)
	open := strings.Index(text, "(")
	closeIdx := strings.LastIndex(text, ")")
	if open <= 0 || closeIdx <= open {
		return "", "", false
	}
	name = strings.TrimSpace(text[:open])
	if name == "" || strings.Contains(name, " ") {
		return "", "", false
	}
	return name, strings.TrimSpace(text[open+1 : closeIdx]), true
}

func stripToolPayloads(text string) (string, []model.AgentChatMessage) {
	var tools []model.AgentChatMessage
	for {
		start := strings.Index(text, `[{"id":`)
		if start < 0 {
			start = strings.Index(text, `[{"id"`)
		}
		if start < 0 {
			break
		}
		end := strings.Index(text[start:], `}]`)
		if end < 0 {
			break
		}
		end = start + end + 2
		chunk := strings.TrimSpace(text[start:end])
		if msg.IsToolCallPayload(chunk) {
			tools = append(tools, toolMessageFromSummary(chunk, time.Time{}))
		}
		text = strings.TrimSpace(text[:start] + text[end:])
	}
	return text, tools
}

func stripRunningToolStatus(text string) string {
	lines := strings.Split(text, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Running tool:") {
			continue
		}
		if strings.HasPrefix(trimmed, "Delegating to subagent:") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func firstToolNameFromPayload(text string) string {
	if name, _, ok := ParseToolSummaryLine(msg.SummarizeToolCallPayload(text)); ok {
		return name
	}
	if name, _, ok := ParseToolSummaryLine(text); ok {
		return name
	}
	return ""
}

// RenderChatAgentMarkdownHTML converts assistant markdown to sanitized HTML for web display.
func RenderChatAgentMarkdownHTML(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	html, err := utils.MarkdownToHTML([]byte(trimmed))
	if err != nil {
		return "<pre class=\"whitespace-pre-wrap font-sans text-sm\">" + htmlEscapeChat(trimmed) + "</pre>"
	}
	safe := string(bluemonday.UGCPolicy().SanitizeBytes(html))
	return enhanceChatAgentMarkdownHTML(safe)
}

func enhanceChatAgentMarkdownHTML(html string) string {
	if html == "" || !strings.Contains(html, "<table") {
		return html
	}
	out := strings.ReplaceAll(html, "<table>", `<div class="chatagent-md-table-wrap"><table>`)
	out = strings.ReplaceAll(out, "<table ", `<div class="chatagent-md-table-wrap"><table `)
	return strings.ReplaceAll(out, "</table>", `</table></div>`)
}

// FormatChatAgentMessageHTML converts assistant markdown to trusted server HTML.
func FormatChatAgentMessageHTML(role, text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	if role != "assistant" {
		return "<pre class=\"whitespace-pre-wrap font-sans text-sm\">" + htmlEscapeChat(trimmed) + "</pre>"
	}
	return RenderChatAgentMarkdownHTML(trimmed)
}

func htmlEscapeChat(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(s)
}
