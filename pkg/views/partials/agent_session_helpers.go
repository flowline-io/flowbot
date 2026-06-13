package partials

import (
	"strings"

	"github.com/a-h/templ"
	"github.com/bytedance/sonic"
)

// agentSessionStateBadgeClass returns DaisyUI badge classes for a display state label.
func agentSessionStateBadgeClass(state string) string {
	switch state {
	case "Active":
		return "badge badge-success badge-sm"
	case "Closed":
		return "badge badge-ghost badge-sm"
	default:
		return "badge badge-warning badge-sm"
	}
}

// agentSessionDetailURL builds the detail page URL for a session flag.
func agentSessionDetailURL(flag string) templ.SafeURL {
	return templ.URL("/service/web/agent-sessions/" + flag)
}

// agentSessionEntryPayloadURL builds the HTMX payload partial URL for an entry.
func agentSessionEntryPayloadURL(sessionID, entryID string) templ.SafeURL {
	return templ.URL("/service/web/agent-sessions/" + sessionID + "/entries/" + entryID + "/payload")
}

// FormatEntryPayload pretty-prints entry payload JSON for display.
func FormatEntryPayload(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	b, err := sonic.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// entryPayloadPreview returns a single-line preview for table cells.
func entryPayloadPreview(payloadJSON string) string {
	if payloadJSON == "" {
		return ""
	}
	const maxLen = 120
	flat := strings.ReplaceAll(strings.ReplaceAll(payloadJSON, "\n", " "), "  ", " ")
	if len(flat) <= maxLen {
		return flat
	}
	return flat[:maxLen] + "..."
}
