package partials

import (
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

// agentSessionDetailURL builds the detail page URL for a session flag.
func agentSessionDetailURL(flag string) templ.SafeURL {
	return templ.URL("/service/web/agent-sessions/" + flag)
}

// AgentSessionThreadURL builds the chatagent thread page URL for a session flag.
func AgentSessionThreadURL(flag string) templ.SafeURL {
	return templ.URL("/service/web/agents/" + strings.TrimSpace(flag))
}

// AgentSessionExportURL builds the session export download URL for a session flag.
func AgentSessionExportURL(flag string) templ.SafeURL {
	return templ.URL("/service/web/agent-sessions/" + strings.TrimSpace(flag) + "/export")
}

// AgentSessionPageTitle returns the browser title for a session detail page.
func AgentSessionPageTitle(session model.AgentSession) string {
	if strings.TrimSpace(session.Title) != "" {
		return session.Title + " — Flowbot"
	}
	return "Session " + session.Flag + " — Flowbot"
}

// AgentResourcePreviewURL builds the HTMX preview URL for a resource URI.
func AgentResourcePreviewURL(sessionID, resourceURI string, full bool) templ.SafeURL {
	q := url.Values{}
	q.Set("uri", resourceURI)
	if full {
		q.Set("full", "1")
	}
	return templ.URL("/service/web/agent-sessions/" + sessionID + "/resources?" + q.Encode())
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
