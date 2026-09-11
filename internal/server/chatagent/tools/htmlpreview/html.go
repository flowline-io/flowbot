// Package htmlpreview provides the chatagent tool for sandboxed HTML artifacts.
package htmlpreview

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"errors"
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

const (
	// ToolName is the agent tool name for presenting HTML artifacts.
	ToolName = "present_html"
	// MaxHTMLBytes rejects html arguments above this size.
	MaxHTMLBytes = 256 << 10
	// DefaultTitle is used when the model omits title.
	DefaultTitle = "HTML"
	// PreviewCSP is injected into the preview document. It is the network lock
	// complementing iframe sandbox="allow-scripts" (without allow-same-origin).
	PreviewCSP = "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"
	maxIDLen   = 64
)

const cspMeta = `<meta http-equiv="Content-Security-Policy" content="` + PreviewCSP + `"><meta name="referrer" content="no-referrer">`

// Tool upserts a session HTML artifact. The full document lives in the tool-call
// arguments; the result is a short stub for the model.
type Tool struct{}

// Name returns the tool identifier.
func (Tool) Name() string { return ToolName }

// Description explains the tool to the model.
func (Tool) Description() string {
	return "Present an interactive HTML document in the chat preview. CSS and JavaScript must be inlined (no network, no CDNs, no fetch). Pass id from a previous result to replace that artifact; omit id to create a new one."
}

// Parameters returns the JSON schema for tool arguments.
func (Tool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"html": map[string]any{
				"type":        "string",
				"description": "Full HTML document or fragment. Must be self-contained: inline CSS and JS only.",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Optional short title shown on the preview card",
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Optional artifact id from a previous present_html result; omit to create a new artifact. Letters, digits, underscore, and hyphen only; max 64 characters.",
			},
		},
		"required": []string{"html"},
	}
}

// Execute validates html, assigns an id when missing, and returns a stub summary.
func (Tool) Execute(_ context.Context, callID string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	parsed, err := ParseArgs(args)
	if err != nil {
		return tool.ErrorResult(callID, ToolName, "invalid_args", err.Error(), "pass html (and optional title/id)"), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: callID,
		Name:       ToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: FormatResult(parsed)}},
	}, nil
}

// ParsedArgs is the validated present_html payload.
type ParsedArgs struct {
	HTML  string
	Title string
	ID    string
	Bytes int
	Hash  string
}

// ParseArgs validates tool arguments and fills id/title defaults.
func ParseArgs(args map[string]any) (ParsedArgs, error) {
	html := stringArg(args["html"])
	if html == "" {
		return ParsedArgs{}, errors.New("html is required")
	}
	if !utf8.ValidString(html) {
		return ParsedArgs{}, errors.New("html must be valid UTF-8")
	}
	if len(html) > MaxHTMLBytes {
		return ParsedArgs{}, fmt.Errorf("html exceeds %d bytes", MaxHTMLBytes)
	}
	title := stringArg(args["title"])
	if title == "" {
		title = DefaultTitle
	}
	id := stringArg(args["id"])
	if id == "" {
		generated, err := newArtifactID()
		if err != nil {
			return ParsedArgs{}, fmt.Errorf("generate artifact id: %w", err)
		}
		id = generated
	} else if err := validateArtifactID(id); err != nil {
		return ParsedArgs{}, err
	}
	return ParsedArgs{
		HTML:  html,
		Title: title,
		ID:    id,
		Bytes: len(html),
		Hash:  HashHTML(html),
	}, nil
}

// HTMLAndTitleFromArguments reads html and title from a tool-call JSON string
// without generating an id or re-validating size (for UI / SSE hydration).
func HTMLAndTitleFromArguments(raw string) (html, title string) {
	var args map[string]any
	if err := sonic.UnmarshalString(strings.TrimSpace(raw), &args); err != nil {
		return "", ""
	}
	html = stringArg(args["html"])
	title = stringArg(args["title"])
	if title == "" {
		title = DefaultTitle
	}
	return html, title
}

// FormatResult builds the stub tool result shown to the model and stored as stdout.
func FormatResult(parsed ParsedArgs) string {
	return fmt.Sprintf("html presented\nid: %s\ntitle: %s\nbytes: %d\nhash: sha256:%s",
		parsed.ID, parsed.Title, parsed.Bytes, parsed.Hash)
}

// ResultMeta is parsed from a present_html stub result.
type ResultMeta struct {
	ID    string
	Title string
}

// ParseResultMeta extracts id/title from a present_html stub result.
func ParseResultMeta(text string) ResultMeta {
	var out ResultMeta
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "id: "):
			out.ID = strings.TrimSpace(strings.TrimPrefix(line, "id: "))
		case strings.HasPrefix(line, "title: "):
			out.Title = strings.TrimSpace(strings.TrimPrefix(line, "title: "))
		}
	}
	return out
}

// HashHTML returns the hex SHA-256 of html (full digest).
func HashHTML(html string) string {
	sum := sha256.Sum256([]byte(html))
	return hex.EncodeToString(sum[:])
}

// PrepareDocument wraps html as a complete document with CSP meta in <head>.
func PrepareDocument(html, title string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		html = "<!-- empty -->"
	}
	if title == "" {
		title = DefaultTitle
	}
	if looksLikeDocument(html) {
		return injectCSP(html, title)
	}
	return wrapFragment(html, title)
}

// Register registers present_html on the given registry.
func Register(registry *tool.Registry) error {
	if registry == nil {
		return errors.New("htmlpreview tools: registry is nil")
	}
	return registry.Register(Tool{})
}

// ActiveToolNames returns the default htmlpreview tool names.
func ActiveToolNames() []string {
	return []string{ToolName}
}

func stringArg(raw any) string {
	if raw == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "<nil>" {
		return ""
	}
	return s
}

func newArtifactID() (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "html_" + hex.EncodeToString(b[:]), nil
}

func validateArtifactID(id string) error {
	if len(id) > maxIDLen {
		return fmt.Errorf("id exceeds %d characters", maxIDLen)
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return errors.New("id must be alphanumeric, underscore, or hyphen")
		}
	}
	return nil
}

func looksLikeDocument(html string) bool {
	trimmed := strings.TrimSpace(html)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "<!doctype") || strings.Contains(lower, "<html")
}

func wrapFragment(html, title string) string {
	return "<!DOCTYPE html><html><head>" + cspMeta +
		"<meta charset=\"utf-8\"><title>" + htmlEscape(title) + "</title></head><body>" + html + "</body></html>"
}

func injectCSP(doc, title string) string {
	lower := strings.ToLower(doc)
	if i, end := indexOpenTag(lower, "head"); i >= 0 && end > i {
		return doc[:end] + cspMeta + ensureTitle(doc[end:], lower[end:], title)
	}
	if i, end := indexOpenTag(lower, "html"); i >= 0 && end > i {
		head := "<head>" + cspMeta + "<meta charset=\"utf-8\"><title>" + htmlEscape(title) + "</title></head>"
		return doc[:end] + head + doc[end:]
	}
	return wrapFragment(doc, title)
}

func ensureTitle(rest, lowerRest, title string) string {
	if strings.Contains(lowerRest, "<title") {
		return rest
	}
	return "<title>" + htmlEscape(title) + "</title>" + rest
}

func indexOpenTag(lower, name string) (start, end int) {
	token := "<" + name
	start = strings.Index(lower, token)
	if start < 0 {
		return -1, -1
	}
	rel := strings.Index(lower[start:], ">")
	if rel < 0 {
		return -1, -1
	}
	return start, start + rel + 1
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
