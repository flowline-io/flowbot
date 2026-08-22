package pages

import (
	"context"
	"strings"
	"unicode"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/i18n"
)

// SettingsValueKind selects how a settings value cell is rendered.
type SettingsValueKind string

const (
	// SettingsValueText is a plain scalar value.
	SettingsValueText SettingsValueKind = "text"
	// SettingsValueSecret is a redacted secret.
	SettingsValueSecret SettingsValueKind = "secret"
	// SettingsValueEmpty is an empty or unset placeholder.
	SettingsValueEmpty SettingsValueKind = "empty"
	// SettingsValueCode is JSON / multi-line structured text.
	SettingsValueCode SettingsValueKind = "code"
)

// SettingsRow is one display row on the Settings page.
type SettingsRow struct {
	Path        string
	Value       string
	Description string
	Sensitive   bool
	FilterText  string
	ValueKind   SettingsValueKind
}

// SettingsSection is a top-level group of settings rows.
type SettingsSection struct {
	Name    string
	Title   string
	Entries []SettingsRow
}

// SettingsPageData holds redacted runtime config sections for the Settings page.
type SettingsPageData struct {
	Sections []SettingsSection
}

// NewSettingsPageData maps a config settings catalog into page view models.
func NewSettingsPageData(ctx context.Context, groups []config.SettingGroup) SettingsPageData {
	sections := make([]SettingsSection, 0, len(groups))
	for _, g := range groups {
		entries := make([]SettingsRow, 0, len(g.Entries))
		for _, e := range g.Entries {
			desc := i18n.TDefault(ctx, config.SettingDescriptionKey(e.Path), e.Description)
			entries = append(entries, SettingsRow{
				Path:        e.Path,
				Value:       e.Value,
				Description: desc,
				Sensitive:   e.Sensitive,
				FilterText:  settingsFilterText(e.Path, e.Value, desc),
				ValueKind:   settingsValueKind(e),
			})
		}
		sections = append(sections, SettingsSection{
			Name:    g.Name,
			Title:   settingsGroupTitle(ctx, g.Name),
			Entries: entries,
		})
	}
	return SettingsPageData{Sections: sections}
}

func settingsFilterText(path, value, description string) string {
	return strings.TrimSpace(path + " " + value + " " + description)
}

func settingsValueKind(e config.SettingEntry) SettingsValueKind {
	switch e.Value {
	case config.EmptyDisplay, config.NotSetDisplay:
		return SettingsValueEmpty
	}
	if e.Sensitive {
		return SettingsValueSecret
	}
	if strings.HasPrefix(e.Value, "[") || strings.HasPrefix(e.Value, "{") || strings.Contains(e.Value, "\n") {
		return SettingsValueCode
	}
	return SettingsValueText
}

func settingsGroupTitle(ctx context.Context, name string) string {
	if name == "" || name == "root" {
		return i18n.T(ctx, "settings.group.root")
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	for i, p := range parts {
		parts[i] = titleCaseWord(p)
	}
	defaultTitle := strings.Join(parts, " ")
	return i18n.TDefault(ctx, "settings.group."+name, defaultTitle)
}

func titleCaseWord(s string) string {
	if s == "" {
		return s
	}
	switch strings.ToLower(s) {
	case "http", "api", "url", "dsn", "ssh", "id":
		return strings.ToUpper(s)
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func settingsSectionCount(ctx context.Context, section SettingsSection) string {
	n := len(section.Entries)
	if n == 1 {
		return i18n.T(ctx, "settings.section.one_key")
	}
	return i18n.TData(ctx, "settings.section.key_count", map[string]any{"Count": n})
}
