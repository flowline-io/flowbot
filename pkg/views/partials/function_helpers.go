package partials

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/flowline-io/flowbot/pkg/functions"
)

// FunctionListEntry is one row on the Functions list page.
type FunctionListEntry struct {
	Name                  string
	Status                string
	Version               int
	PublishedVersion      *int
	HasUnpublishedChanges bool
}

// FunctionDraftData is the editor page model (secrets already redacted).
type FunctionDraftData struct {
	Name                  string
	Status                string
	Version               int
	Entrypoint            string
	Source                string
	EnvText               string
	Token                 string
	TokenSet              bool
	HMACSecret            string
	HMACSet               bool
	PublishedVersion      *int
	HasUnpublishedChanges bool
	CallURL               string
	CallVersionURL        string
}

// FunctionWebPath returns the editor URL for a function name.
func FunctionWebPath(name string) string {
	return "/service/web/functions/" + url.PathEscape(name)
}

// FunctionCallPath returns the HTTP invoke path for a named function (latest published).
func FunctionCallPath(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return "/service/functions/call/" + url.PathEscape(name)
}

// FunctionCallVersionPath returns the HTTP invoke path for a specific published version.
func FunctionCallVersionPath(name string, version int) string {
	path := FunctionCallPath(name)
	if path == "" || version <= 0 {
		return ""
	}
	return path + "/v/" + fmt.Sprintf("%d", version)
}

// FunctionCallURL returns the absolute call URL when publicOrigin is set, otherwise the path.
func FunctionCallURL(name, publicOrigin string) string {
	path := FunctionCallPath(name)
	if path == "" {
		return ""
	}
	base := strings.TrimRight(strings.TrimSpace(publicOrigin), "/")
	if base == "" {
		return path
	}
	return base + path
}

// FunctionCallVersionURL returns the absolute or path call URL for a published version.
func FunctionCallVersionURL(name string, version int, publicOrigin string) string {
	path := FunctionCallVersionPath(name, version)
	if path == "" {
		return ""
	}
	base := strings.TrimRight(strings.TrimSpace(publicOrigin), "/")
	if base == "" {
		return path
	}
	return base + path
}

// BuildFunctionListEntries maps service ListAll rows to list entries.
func BuildFunctionListEntries(items []functions.ListAllInfo) []FunctionListEntry {
	out := make([]FunctionListEntry, 0, len(items))
	for _, item := range items {
		out = append(out, FunctionListEntry{
			Name:                  item.Name,
			Status:                item.Status,
			Version:               item.Version,
			PublishedVersion:      item.PublishedVersion,
			HasUnpublishedChanges: item.HasUnpublishedChanges,
		})
	}
	return out
}

// FunctionDraftFromView maps a redacted DraftView to editor data.
func FunctionDraftFromView(view *functions.DraftView) FunctionDraftData {
	if view == nil {
		return FunctionDraftData{}
	}
	return FunctionDraftData{
		Name:                  view.Name,
		Status:                view.Status,
		Version:               view.Version,
		Entrypoint:            view.Entrypoint,
		Source:                view.Source,
		EnvText:               formatEnvText(view.Env),
		Token:                 view.Token,
		TokenSet:              view.TokenSet,
		HMACSecret:            view.HMACSecret,
		HMACSet:               view.HMACSet,
		PublishedVersion:      view.PublishedVersion,
		HasUnpublishedChanges: view.HasUnpublishedChanges,
	}
}

func formatEnvText(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+env[k])
	}
	return strings.Join(lines, "\n")
}

// PublishedVersionAttr returns the published version for data attributes.
func PublishedVersionAttr(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

// FormatFunctionRunDuration formats run duration for display.
func FormatFunctionRunDuration(ms int64) string {
	if ms <= 0 {
		return "—"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}
