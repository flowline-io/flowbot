package partials

import (
	"fmt"
	"hash/fnv"
	"html"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

// EventTypeChipClass returns the flowbot-chip CSS class for an event type.
func EventTypeChipClass(eventType string) string {
	lower := strings.ToLower(eventType)
	switch {
	case lower == "":
		return "flowbot-chip flowbot-chip-muted"
	case strings.HasSuffix(lower, ".failed"), strings.Contains(lower, "error"):
		return "flowbot-chip flowbot-chip-error"
	case strings.HasSuffix(lower, ".created"), strings.HasSuffix(lower, ".success"):
		return "flowbot-chip flowbot-chip-success"
	case strings.HasPrefix(lower, "webhook."):
		return "flowbot-chip flowbot-chip-primary"
	default:
		return "flowbot-chip flowbot-chip-muted"
	}
}

var eventSourceChipPalette = []string{
	"flowbot-chip flowbot-chip-primary",
	"flowbot-chip flowbot-chip-muted",
	"flowbot-chip flowbot-chip-warning",
}

// EventSourceChipClass returns a stable chip class for a source name.
func EventSourceChipClass(source string) string {
	if source == "" {
		return "flowbot-chip flowbot-chip-muted"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(source))
	return eventSourceChipPalette[h.Sum32()%uint32(len(eventSourceChipPalette))]
}

// EventRunStatusChipClass returns the chip class for a pipeline run status string.
func EventRunStatusChipClass(status string) string {
	switch status {
	case "2":
		return "flowbot-chip flowbot-chip-success"
	case "4":
		return "flowbot-chip flowbot-chip-error"
	case "3":
		return "flowbot-chip flowbot-chip-warning"
	default:
		return "flowbot-chip flowbot-chip-muted"
	}
}

// EventRunStatusText returns a short label for a pipeline run status string.
func EventRunStatusText(status string) string {
	switch status {
	case "2":
		return "Success"
	case "4":
		return "Failed"
	case "3":
		return "Cancelled"
	case "1":
		return "Running"
	default:
		return "Started"
	}
}

// WebhookMethodChipClass returns the chip class for an HTTP method.
func WebhookMethodChipClass(method string) string {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH":
		return "flowbot-chip flowbot-chip-primary"
	case "DELETE":
		return "flowbot-chip flowbot-chip-error"
	default:
		return "flowbot-chip flowbot-chip-muted"
	}
}

// SimilarEventsURL builds the events page URL filtered by source and/or type.
func SimilarEventsURL(source, eventType string) string {
	q := url.Values{}
	if source != "" {
		q.Set("source", source)
	}
	if eventType != "" {
		q.Set("type", eventType)
	}
	if len(q) == 0 {
		return "/service/web/events"
	}
	return "/service/web/events?" + q.Encode()
}

// PipelineRunLiveURL builds the live run page URL for a pipeline run.
func PipelineRunLiveURL(pipelineName string, runID int64) string {
	return fmt.Sprintf("/service/web/pipelines/%s/runs/%d/live",
		url.PathEscape(pipelineName), runID)
}

// PrettyJSON indents compact JSON; empty input becomes "{}"; invalid JSON is returned unchanged.
func PrettyJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	var v any
	if err := sonic.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	b, err := sonic.MarshalIndent(v, "", "  ")
	if err != nil {
		return raw
	}
	return string(b)
}

// HighlightJSON wraps JSON tokens in span classes for syntax highlighting.
// All text is HTML-escaped. Invalid JSON is returned fully escaped without spans.
func HighlightJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	var v any
	if err := sonic.Unmarshal([]byte(raw), &v); err != nil {
		return html.EscapeString(raw)
	}
	return highlightJSONValue(v, 0)
}

func highlightJSONValue(v any, indent int) string {
	switch t := v.(type) {
	case map[string]any:
		return highlightJSONObject(t, indent)
	case []any:
		return highlightJSONArray(t, indent)
	case string:
		return jsonSpan("flowbot-json-string", html.EscapeString(strconv.Quote(t)))
	case bool:
		return jsonSpan("flowbot-json-bool", strconv.FormatBool(t))
	case nil:
		return jsonSpan("flowbot-json-null", "null")
	case float64:
		return jsonSpan("flowbot-json-number", formatJSONNumber(t))
	case int64:
		return jsonSpan("flowbot-json-number", strconv.FormatInt(t, 10))
	default:
		return highlightJSONFallback(t)
	}
}

func highlightJSONObject(m map[string]any, indent int) string {
	if len(m) == 0 {
		return "{}"
	}
	pad := strings.Repeat("  ", indent)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		line := pad + "  " +
			jsonSpan("flowbot-json-key", html.EscapeString(strconv.Quote(k))) +
			": " + highlightJSONValue(m[k], indent+1)
		lines = append(lines, line)
	}
	return "{\n" + strings.Join(lines, ",\n") + "\n" + pad + "}"
}

func highlightJSONArray(items []any, indent int) string {
	if len(items) == 0 {
		return "[]"
	}
	pad := strings.Repeat("  ", indent)
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, pad+"  "+highlightJSONValue(item, indent+1))
	}
	return "[\n" + strings.Join(lines, ",\n") + "\n" + pad + "]"
}

func highlightJSONFallback(v any) string {
	raw, err := sonic.Marshal(v)
	if err != nil {
		return html.EscapeString(fmt.Sprint(v))
	}
	s := string(raw)
	switch s {
	case "true", "false":
		return jsonSpan("flowbot-json-bool", s)
	case "null":
		return jsonSpan("flowbot-json-null", "null")
	}
	if len(s) > 0 && s[0] == '"' {
		return jsonSpan("flowbot-json-string", html.EscapeString(s))
	}
	if len(s) > 0 && (s[0] == '{' || s[0] == '[') {
		return html.EscapeString(s)
	}
	return jsonSpan("flowbot-json-number", html.EscapeString(s))
}

func jsonSpan(class, content string) string {
	return `<span class="` + class + `">` + content + `</span>`
}

func formatJSONNumber(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
