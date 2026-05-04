// Package template provides notification template rendering using Go text/template
// with Sprig function library support.
package template

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/flowline-io/flowbot/pkg/config"
)

// Engine renders notification templates for different channels using Sprig functions.
type Engine struct {
	templates map[string]*templateEntry
}

type templateEntry struct {
	manifest *config.NotifyTemplate
	compiled map[string]*template.Template // keyed by channel ("" for default)
}

// New creates a new template Engine.
func New() *Engine {
	return &Engine{
		templates: make(map[string]*templateEntry),
	}
}

// LoadConfig loads templates from notify configuration and compiles all templates.
func (e *Engine) LoadConfig(templates []config.NotifyTemplate) error {
	for i := range templates {
		tmpl := &templates[i]
		entry := &templateEntry{
			manifest: tmpl,
			compiled: make(map[string]*template.Template),
		}

		// compile default template
		compiled, err := compileTemplate(tmpl.DefaultTemplate, tmpl.DefaultFormat)
		if err != nil {
			return err
		}
		entry.compiled[""] = compiled

		// compile channel overrides
		for _, override := range tmpl.Overrides {
			compiled, err := compileTemplate(override.Template, override.Format)
			if err != nil {
				return err
			}
			entry.compiled[override.Channel] = compiled
		}

		e.templates[tmpl.ID] = entry
	}
	return nil
}

// RenderResult holds the output of template rendering for a specific channel.
type RenderResult struct {
	Title  string
	Body   string
	Format string
}

// Render renders a template by ID for a specific channel with the given data payload.
// It first looks for a channel-specific override, falling back to the default template.
func (e *Engine) Render(templateID, channel string, data map[string]any) (*RenderResult, error) {
	entry, ok := e.templates[templateID]
	if !ok {
		return nil, nil
	}

	// use channel override if available
	compiled, ok := entry.compiled[channel]
	if !ok {
		compiled = entry.compiled[""]
	}

	format := entry.manifest.DefaultFormat
	for _, o := range entry.manifest.Overrides {
		if o.Channel == channel {
			format = o.Format
			break
		}
	}

	var buf bytes.Buffer
	if err := compiled.Execute(&buf, data); err != nil {
		return nil, err
	}

	body := buf.String()
	title := extractTitle(body)

	return &RenderResult{
		Title:  title,
		Body:   body,
		Format: format,
	}, nil
}

// GetTemplateID returns the template ID for a given event type.
// This is a convenience mapping that can be extended.
func (e *Engine) GetTemplateID(eventType string) string {
	// direct lookup
	if _, ok := e.templates[eventType]; ok {
		return eventType
	}
	return ""
}

func compileTemplate(tmplStr string, format string) (*template.Template, error) {
	tmpl := template.New("notify").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"eventTime": eventTime,
		"shorten":   shorten,
	})

	return tmpl.Parse(tmplStr)
}

// extractTitle extracts the first line from a rendered body to use as a title.
func extractTitle(body string) string {
	if idx := strings.IndexByte(body, '\n'); idx > 0 {
		title := strings.TrimSpace(body[:idx])
		// strip markdown formatting for plain title
		title = strings.TrimPrefix(title, "# ")
		title = strings.TrimPrefix(title, "**")
		title = strings.TrimSuffix(title, "**")
		return title
	}
	return body
}

// eventTime formats a time string for display.
func eventTime(t interface{}) string {
	switch v := t.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// shorten truncates a string to a maximum length, appending "..." when truncated.
// The minimum effective maxLen is 4 (to leave room for the ellipsis).
func shorten(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		maxLen = 4
	}
	return s[:maxLen-3] + "..."
}
