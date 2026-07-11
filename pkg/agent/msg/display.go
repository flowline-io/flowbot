package msg

import (
	"regexp"
	"strings"

	"github.com/bytedance/sonic"
)

const toolCallDisplayMaxLen = 320

type functionCallFunction struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

type functionCallPayload struct {
	ID       string                `json:"id"`
	Type     string                `json:"type"`
	Name     string                `json:"name"`
	Function *functionCallFunction `json:"function"`
}

var (
	toolFunctionBlockPattern = regexp.MustCompile(`"function"\s*:\s*\{[^}]*"name"\s*:\s*"([^"]+)"`)
	toolNamePattern          = regexp.MustCompile(`"name"\s*:\s*"([^"]+)"`)
)

// AssistantDisplayText renders user-visible assistant content from a message.
func AssistantDisplayText(m AssistantMessage) string {
	if calls := m.ToolCalls(); len(calls) > 0 {
		if summary := SummarizeToolCallParts(calls); summary != "" {
			return summary
		}
	}
	raw := strings.TrimSpace(m.TextContent())
	if raw == "" {
		return ""
	}
	return strings.TrimSpace(SanitizeAssistantDisplayText(raw))
}

// SummarizeToolCallParts renders structured tool calls as a one-line summary.
func SummarizeToolCallParts(calls []ToolCallPart) string {
	payloads := make([]functionCallPayload, 0, len(calls))
	for _, call := range calls {
		name := strings.TrimSpace(call.Name)
		if name == "" {
			continue
		}
		payloads = append(payloads, functionCallPayload{
			ID:   call.ID,
			Type: "function",
			Function: &functionCallFunction{
				Name:      name,
				Arguments: call.Arguments,
			},
		})
	}
	if len(payloads) == 0 {
		return ""
	}
	return truncateOneLine(joinToolCallSummaries(payloads), toolCallDisplayMaxLen)
}

// SanitizeAssistantDisplayText returns user-visible assistant text, collapsing
// raw tool-call JSON into a one-line summary.
func SanitizeAssistantDisplayText(text string) string {
	if IsToolCallPayload(text) {
		if summary := SummarizeToolCallPayload(text); summary != "" {
			return summary
		}
		return "Running tool..."
	}
	return text
}

// SummarizeToolCallPayload renders tool-call JSON as a compact one-line summary.
func SummarizeToolCallPayload(text string) string {
	text = strings.TrimSpace(text)
	if calls := parseToolCallsLoose(text); len(calls) > 0 {
		return truncateOneLine(joinToolCallSummaries(calls), toolCallDisplayMaxLen)
	}
	if summary := summarizePartialToolCall(text); summary != "" {
		return truncateOneLine(summary, toolCallDisplayMaxLen)
	}
	return ""
}

// IsToolCallStreamDelta reports whether one streaming chunk carries tool-call JSON
// emitted by langchaingo instead of user-visible assistant text.
func IsToolCallStreamDelta(delta string) bool {
	delta = strings.TrimSpace(delta)
	if delta == "" {
		return false
	}
	prefixes := []string{
		`[{"id":`,
		`[{"id" :`,
		`[{"type":"function"`,
		`[{"type": "function"`,
		`[{"type":"","function"`,
		`[{"type": "","function"`,
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(delta, prefix) {
			return true
		}
	}
	if strings.HasPrefix(delta, `{"name":`) && strings.Contains(delta, `"arguments"`) {
		return true
	}
	if strings.HasPrefix(delta, `{"name" :`) && strings.Contains(delta, `"arguments"`) {
		return true
	}
	return false
}

// TrimToolCallStreamContent removes langchaingo tool-call JSON from assistant text,
// keeping any preceding user-visible content.
func TrimToolCallStreamContent(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if IsToolCallOnlyContent(text) {
		return ""
	}
	markers := []string{
		`[{"id":`,
		`[{"type":"function"`,
		`[{"type": "function"`,
		`[{"type":"","function"`,
	}
	cut := -1
	for _, marker := range markers {
		if idx := strings.Index(text, marker); idx >= 0 && (cut < 0 || idx < cut) {
			cut = idx
		}
	}
	if cut <= 0 {
		return text
	}
	return strings.TrimSpace(text[:cut])
}

// IsToolCallOnlyContent reports whether text is exclusively tool-call JSON with
// no preceding user-visible assistant text.
func IsToolCallOnlyContent(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if isToolCallPrefix(text) {
		return true
	}
	if strings.HasPrefix(text, `{`) {
		return IsToolCallPayload(text)
	}
	return false
}

// IsToolCallPayload reports whether text is a serialized tool invocation payload.
func IsToolCallPayload(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if isToolCallPrefix(text) {
		return true
	}
	if strings.Contains(text, `"type":"function"`) || strings.Contains(text, `"type": "function"`) {
		return strings.Contains(text, `"function"`)
	}
	calls, ok := parseToolCalls(text)
	return ok && len(calls) > 0 && allLookLikeToolCalls(calls)
}

func parseToolCallsLoose(text string) []functionCallPayload {
	candidates := make([]functionCallPayload, 0, 4)
	if calls, ok := parseToolCalls(text); ok && len(calls) > 0 {
		for _, call := range calls {
			if looksLikeToolCallObject(call) {
				candidates = append(candidates, enrichCallArguments(call, text))
			}
		}
	}
	for _, chunk := range extractJSONObjectChunks(text) {
		var call functionCallPayload
		if err := sonic.UnmarshalString(chunk, &call); err != nil {
			continue
		}
		if !looksLikeToolCallObject(call) {
			continue
		}
		candidates = append(candidates, enrichCallArguments(call, chunk))
	}
	return dedupeToolCalls(candidates)
}

func dedupeToolCalls(calls []functionCallPayload) []functionCallPayload {
	best := make(map[string]functionCallPayload, len(calls))
	order := make([]string, 0, len(calls))
	for _, call := range calls {
		name := toolCallName(call)
		if name == "" {
			continue
		}
		prev, ok := best[name]
		if !ok {
			order = append(order, name)
			best[name] = call
			continue
		}
		if toolArgumentScore(call) > toolArgumentScore(prev) {
			best[name] = call
		}
	}
	out := make([]functionCallPayload, 0, len(order))
	for _, name := range order {
		out = append(out, best[name])
	}
	return out
}

func toolArgumentScore(call functionCallPayload) int {
	args := strings.TrimSpace(toolCallArguments(call))
	if !isDisplayableToolArguments(args) {
		return 0
	}
	return len(args)
}

func enrichCallArguments(call functionCallPayload, source string) functionCallPayload {
	if isDisplayableToolArguments(toolCallArguments(call)) {
		return call
	}
	args := extractToolCallArgumentsValue(source)
	if !isDisplayableToolArguments(args) {
		return call
	}
	if call.Function == nil {
		call.Function = &functionCallFunction{Name: toolCallName(call)}
	}
	call.Function.Arguments = args
	return call
}

func extractJSONObjectChunks(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	chunks := make([]string, 0, 4)
	for i := 0; i < len(text); {
		start := strings.Index(text[i:], "{")
		if start < 0 {
			break
		}
		start += i
		end := matchingBraceEnd(text, start)
		if end < 0 {
			break
		}
		chunks = append(chunks, text[start:end+1])
		i = end + 1
	}
	return chunks
}

func matchingBraceEnd(text string, start int) int {
	if start < 0 || start >= len(text) || text[start] != '{' {
		return -1
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if escape {
			escape = false
			continue
		}
		if inString {
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func parseToolCalls(text string) ([]functionCallPayload, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	if strings.HasPrefix(text, "[") {
		var calls []functionCallPayload
		if err := sonic.UnmarshalString(text, &calls); err != nil {
			return nil, false
		}
		return calls, true
	}
	if strings.HasPrefix(text, "{") {
		var call functionCallPayload
		if err := sonic.UnmarshalString(text, &call); err != nil {
			return nil, false
		}
		return []functionCallPayload{call}, true
	}
	return nil, false
}

func joinToolCallSummaries(calls []functionCallPayload) string {
	summaries := make([]string, 0, len(calls))
	for _, call := range calls {
		summary := summarizeOneToolCall(call)
		if summary == "" {
			continue
		}
		summaries = append(summaries, summary)
	}
	return strings.Join(summaries, ", ")
}

func summarizeOneToolCall(call functionCallPayload) string {
	name := toolCallName(call)
	if name == "" {
		return ""
	}
	args, ok := formatToolCallArgsForDisplay(toolCallArguments(call))
	if !ok {
		return name + "(...)"
	}
	return name + "(" + args + ")"
}

func summarizePartialToolCall(text string) string {
	name := extractToolCallName(text)
	if name == "" {
		return ""
	}
	args, ok := formatToolCallArgsForDisplay(extractToolCallArgumentsValue(text))
	if !ok {
		return name + "(...)"
	}
	return name + "(" + args + ")"
}

func formatToolCallArgsForDisplay(args string) (string, bool) {
	args = collapseToOneLine(args, 0)
	if !isDisplayableToolArguments(args) {
		return "", false
	}
	return args, true
}

func isDisplayableToolArguments(args string) bool {
	args = strings.TrimSpace(args)
	if args == "" || args == `""` || args == `"` {
		return false
	}
	if strings.Contains(args, `"}}`) {
		return false
	}
	if strings.HasSuffix(args, "}}") && !strings.HasPrefix(args, "{") {
		return false
	}
	if sonic.ValidString(args) {
		var value any
		if err := sonic.UnmarshalString(args, &value); err != nil {
			return false
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			return false
		}
		return true
	}
	return strings.HasPrefix(args, "{") || strings.HasPrefix(args, "[")
}

func extractToolCallArgumentsValue(text string) string {
	key := `"arguments"`
	_, after, ok := strings.Cut(text, key)
	if !ok {
		return ""
	}
	rest := strings.TrimSpace(after)
	if strings.HasPrefix(rest, ":") {
		rest = strings.TrimSpace(rest[1:])
	}
	return parseLeadingJSONValue(rest)
}

func parseLeadingJSONValue(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	switch text[0] {
	case '"':
		var value string
		if err := sonic.UnmarshalString(text, &value); err == nil {
			return value
		}
		return parsePartialJSONString(text)
	case '{':
		end := matchingBraceEnd(text, 0)
		if end >= 0 {
			return text[:end+1]
		}
		return ""
	default:
		return ""
	}
}

func parsePartialJSONString(text string) string {
	if !strings.HasPrefix(text, `"`) {
		return ""
	}
	var b strings.Builder
	escape := false
	for i := 1; i < len(text); i++ {
		ch := text[i]
		if escape {
			_ = b.WriteByte(ch)
			escape = false
			continue
		}
		if ch == '\\' {
			escape = true
			continue
		}
		if ch == '"' {
			return b.String()
		}
		_ = b.WriteByte(ch)
	}
	return ""
}

func extractToolCallName(text string) string {
	if match := toolFunctionBlockPattern.FindStringSubmatch(text); len(match) >= 2 {
		return match[1]
	}
	fnIdx := strings.Index(text, `"function"`)
	if fnIdx >= 0 {
		if match := toolNamePattern.FindStringSubmatch(text[fnIdx:]); len(match) >= 2 {
			return match[1]
		}
	}
	return ""
}

func toolCallName(call functionCallPayload) string {
	if call.Function != nil && strings.TrimSpace(call.Function.Name) != "" {
		return call.Function.Name
	}
	if strings.TrimSpace(call.Name) != "" {
		return call.Name
	}
	return ""
}

func toolCallArguments(call functionCallPayload) string {
	if call.Function == nil || call.Function.Arguments == nil {
		return ""
	}
	switch v := call.Function.Arguments.(type) {
	case string:
		return v
	default:
		out, err := sonic.MarshalString(v)
		if err != nil {
			return ""
		}
		return out
	}
}

func collapseToOneLine(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	return truncateOneLine(text, maxLen)
}

func truncateOneLine(text string, maxLen int) string {
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}
	const suffix = "..."
	if maxLen <= len(suffix) {
		return text[:maxLen]
	}
	return text[:maxLen-len(suffix)] + suffix
}

func isToolCallPrefix(text string) bool {
	prefixes := []string{
		`[{"id":`,
		`[{"id" :`,
		`[{"type":"function"`,
		`[{"type": "function"`,
		`[{"type":"function",`,
		`[{"type": "function",`,
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

func allLookLikeToolCalls(calls []functionCallPayload) bool {
	for _, call := range calls {
		if !looksLikeToolCallObject(call) {
			return false
		}
	}
	return true
}

func looksLikeToolCallObject(call functionCallPayload) bool {
	if toolCallName(call) != "" {
		return true
	}
	return strings.HasPrefix(call.ID, "call_")
}

// CoalesceAssistantHistoryMessages merges consecutive assistant tool-call
// snapshots, keeping the richest display text for each run.
func CoalesceAssistantHistoryMessages(texts []string) []string {
	if len(texts) == 0 {
		return nil
	}
	out := make([]string, 0, len(texts))
	for _, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if len(out) == 0 {
			out = append(out, text)
			continue
		}
		prev := out[len(out)-1]
		if !IsAssistantToolSummary(prev) || !IsAssistantToolSummary(text) {
			out = append(out, text)
			continue
		}
		if toolSummaryScore(text) >= toolSummaryScore(prev) {
			out[len(out)-1] = text
		}
	}
	return out
}

// IsAssistantToolSummary reports whether text is a compact tool-call display line.
func IsAssistantToolSummary(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if IsToolCallPayload(text) {
		return true
	}
	open := strings.Index(text, "(")
	if open <= 0 {
		return false
	}
	name := strings.TrimSpace(text[:open])
	if name == "" || strings.Contains(name, " ") {
		return false
	}
	return strings.HasSuffix(text, ")")
}

func isToolCallSummary(text string) bool {
	return IsAssistantToolSummary(text)
}

func toolSummaryScore(text string) int {
	summary := text
	if IsToolCallPayload(text) {
		summary = SummarizeToolCallPayload(text)
	}
	if found := strings.Contains(summary, "("); found && strings.HasSuffix(summary, ")") {
		return len(summary)
	}
	return len(summary)
}
