package partials

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

// ChatAgentSessionListURL builds the HTMX list URL including optional filter and cursor.
func ChatAgentSessionListURL(endpoints ChatAgentEndpoints, cursor string) string {
	base := strings.TrimSpace(endpoints.ListURL)
	if base == "" {
		base = "/service/web/agents/list"
	}
	q := make([]string, 0, 2)
	if filter := strings.TrimSpace(endpoints.Filter); filter != "" {
		q = append(q, "filter="+filter)
	}
	if cursor != "" {
		q = append(q, "cursor="+cursor)
	}
	if len(q) == 0 {
		return base
	}
	return base + "?" + strings.Join(q, "&")
}

// ChatAgentListActionURL builds pin/archive action URLs and preserves the active list filter.
func ChatAgentListActionURL(template, sessionID, filter string) string {
	url := ChatAgentDetailURL(template, sessionID)
	if f := strings.TrimSpace(filter); f != "" {
		return url + "?filter=" + f
	}
	return url
}

func chatAgentSessionDisplayState(item model.AgentSession) string {
	switch item.Activity {
	case "needs_approval":
		return "NeedsApproval"
	case "running":
		return "Running"
	default:
		return item.State
	}
}

const chatAgentSessionPreviewLimit = 96

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

// TruncateChatAgentSessionPreview shortens last-message text for list rows.
func TruncateChatAgentSessionPreview(text string, limit int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return ""
	}
	if limit <= 0 {
		limit = chatAgentSessionPreviewLimit
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit == 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}

// ChatAgentSessionActivityLabel returns a human-readable runtime activity label.
func ChatAgentSessionActivityLabel(activity string) string {
	switch strings.TrimSpace(activity) {
	case "running":
		return "Running"
	case "needs_approval":
		return "Needs approval"
	default:
		return ""
	}
}

// GroupAgentSessionsForList builds pinned + calendar-day buckets for the agents home list.
func GroupAgentSessionsForList(items []model.AgentSession, now time.Time) []model.AgentSessionDayGroup {
	if len(items) == 0 {
		return nil
	}
	pinned := make([]model.AgentSession, 0)
	rest := make([]model.AgentSession, 0, len(items))
	for _, item := range items {
		if item.Pinned {
			pinned = append(pinned, item)
			continue
		}
		rest = append(rest, item)
	}
	groups := make([]model.AgentSessionDayGroup, 0, 4)
	if len(pinned) > 0 {
		groups = append(groups, model.AgentSessionDayGroup{
			Key:   "pinned",
			Label: "Pinned",
			Items: pinned,
		})
	}
	groups = append(groups, GroupAgentSessionsByDay(rest, now)...)
	return groups
}

// GroupAgentSessionsByDay buckets sessions by local calendar day of UpdatedAt.
// Input order is preserved within each day group.
func GroupAgentSessionsByDay(items []model.AgentSession, now time.Time) []model.AgentSessionDayGroup {
	if len(items) == 0 {
		return nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)

	groups := make([]model.AgentSessionDayGroup, 0)
	indexByKey := make(map[string]int, 8)
	for _, item := range items {
		at := item.UpdatedAt.In(loc)
		day := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, loc)
		key := day.Format("2006-01-02")
		label := key
		switch {
		case day.Equal(today):
			key = "today"
			label = "Today"
		case day.Equal(yesterday):
			key = "yesterday"
			label = "Yesterday"
		default:
			label = day.Format("Mon, Jan 2")
		}
		if idx, ok := indexByKey[key]; ok {
			groups[idx].Items = append(groups[idx].Items, item)
			continue
		}
		indexByKey[key] = len(groups)
		groups = append(groups, model.AgentSessionDayGroup{
			Key:   key,
			Label: label,
			Items: []model.AgentSession{item},
		})
	}
	return groups
}

// FormatChatAgentRelativeTime returns a compact relative timestamp (e.g. "2h", "6d").
func FormatChatAgentRelativeTime(t time.Time) string {
	return chatAgentRelativeTimeSince(t, time.Now())
}

// ChatAgentToolCardExpanded reports whether a tool card should render expanded.
// Successful and in-progress cards stay collapsed; failures and approval gates open.
func ChatAgentToolCardExpanded(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "error", "failed", "needs_approval":
		return true
	default:
		return false
	}
}

// chatAgentPendingConfirmMeta formats permission / pattern metadata for the approval panel.
func chatAgentPendingConfirmMeta(pending *ChatAgentPendingConfirm) string {
	if pending == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if p := strings.TrimSpace(pending.Permission); p != "" {
		parts = append(parts, "permission: "+p)
	}
	if p := strings.TrimSpace(pending.Pattern); p != "" {
		parts = append(parts, "pattern: "+p)
	}
	return strings.Join(parts, " · ")
}

// ChatAgentApproveOnceLabel is the primary one-shot approval button text.
func ChatAgentApproveOnceLabel() string {
	return "Allow once"
}

// ChatAgentApproveOnceHint explains that once applies only to this tool call.
func ChatAgentApproveOnceHint() string {
	return "This tool call only"
}

// ChatAgentApproveAlwaysLabel is the remember-pattern approval button text.
func ChatAgentApproveAlwaysLabel() string {
	return "Always allow matching"
}

// ChatAgentApproveAlwaysHint explains that always remembers the suggested pattern for future matching calls.
func ChatAgentApproveAlwaysHint(suggestedPattern string) string {
	pattern := strings.TrimSpace(suggestedPattern)
	if pattern == "" {
		return "Remember this pattern for future matching calls"
	}
	return "Remember for future matching calls: " + pattern
}

// ChatAgentApproveDenyLabel is the reject button text.
func ChatAgentApproveDenyLabel() string {
	return "Deny"
}

// FormatPendingApprovalBadgeText formats a compact count for nav / home badges.
// Zero returns empty so callers can hide the badge.
func FormatPendingApprovalBadgeText(count int) string {
	if count <= 0 {
		return ""
	}
	if count > 99 {
		return "99+"
	}
	return strconv.Itoa(count)
}

// ChatAgentPendingConfirmFromEvent maps a stream confirm payload into view data.
func ChatAgentPendingConfirmFromEvent(pending ChatAgentPendingConfirm) *ChatAgentPendingConfirm {
	pending.ID = strings.TrimSpace(pending.ID)
	if pending.ID == "" {
		return nil
	}
	cp := pending
	return &cp
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
	switch state {
	case "Active", "Running", "NeedsApproval":
		return base + " agents-session-thumb-active"
	default:
		return base
	}
}

// chatAgentSessionBadgeClass returns pill badge classes for the session thumbnail.
func chatAgentSessionBadgeClass(state string) string {
	switch state {
	case "Active":
		return "agents-session-badge agents-session-badge-active"
	case "Running":
		return "agents-session-badge agents-session-badge-running"
	case "NeedsApproval":
		return "agents-session-badge agents-session-badge-needs-approval"
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

// chatAgentModelLabel returns the display label for the current session model.
func chatAgentModelLabel(session model.AgentSession, defaultModel string) string {
	m := strings.TrimSpace(session.Model)
	if m == "" {
		m = defaultModel
	}
	if m == "" {
		return ""
	}
	return m
}

// chatAgentSelectedModel returns the model id to preselect in the picker.
func chatAgentSelectedModel(storedModel, defaultModel string) string {
	if m := strings.TrimSpace(storedModel); m != "" {
		return m
	}
	return strings.TrimSpace(defaultModel)
}

// chatAgentModelMultimodal reports whether the selected (or first) model accepts media input.
func chatAgentModelMultimodal(models []SelectableModelOption, selectedModel string) bool {
	selected := strings.TrimSpace(selectedModel)
	if selected != "" {
		for _, m := range models {
			if m.ID == selected {
				return m.Multimodal
			}
		}
	}
	if len(models) > 0 {
		return models[0].Multimodal
	}
	return false
}

// chatAgentMultimodalAttr returns "true" or "false" for HTML data attributes.
func chatAgentMultimodalAttr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// chatAgentSessionSettingsLabel returns the header line for model and thinking level.
func chatAgentSessionSettingsLabel(session model.AgentSession, defaultModel string) string {
	modelName := chatAgentModelLabel(session, defaultModel)
	thinking := chatAgentThinkingLabel(session.ThinkingLevel)
	switch {
	case modelName != "" && thinking != "":
		return modelName + " · Thinking: " + thinking
	case modelName != "":
		return modelName
	case thinking != "":
		return "Thinking: " + thinking
	default:
		return ""
	}
}

// chatAgentThinkingLabel returns a human-readable thinking level label.
func chatAgentThinkingLabel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "off":
		return "Off"
	case "low":
		return "Low"
	case "medium":
		return "Medium"
	case "high":
		return "High"
	default:
		return "Default"
	}
}

// chatAgentThinkingSelected reports whether value matches the session thinking level.
func chatAgentThinkingSelected(value, selectedThinking string) bool {
	normalized := strings.ToLower(strings.TrimSpace(selectedThinking))
	if normalized == "" {
		normalized = "default"
	}
	return value == normalized
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
	html, err := utils.MarkdownToSafeHTML([]byte(trimmed))
	if err != nil {
		return "<pre class=\"whitespace-pre-wrap font-sans text-sm\">" + htmlEscapeChat(trimmed) + "</pre>"
	}
	return enhanceChatAgentMarkdownHTML(string(html))
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
