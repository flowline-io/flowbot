package web

import (
	"net/url"
	"strings"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

const commandPaletteMaxPerGroup = 8

// commandPaletteRecentMax is the max recent visits kept in localStorage (mirrored in JS).
const commandPaletteRecentMax = 8

// commandPaletteRecentStorageKey is the localStorage key for recent visits (mirrored in JS).
const commandPaletteRecentStorageKey = "flowbot-command-palette-recent"

// commandPaletteItem is one jump target in the global command palette.
type commandPaletteItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Href     string `json:"href"`
	Group    string `json:"group"`
}

// commandPaletteResults groups search hits for the command palette JSON API.
type commandPaletteResults struct {
	Pages     []commandPaletteItem `json:"pages"`
	Pipelines []commandPaletteItem `json:"pipelines"`
	Sessions  []commandPaletteItem `json:"sessions"`
	Homelab   []commandPaletteItem `json:"homelab"`
}

// buildCommandPaletteResults filters nav pages, pipelines, sessions, and homelab
// apps by a case-insensitive substring query. Empty/whitespace q yields empty groups.
func buildCommandPaletteResults(
	q string,
	pages []commandPaletteItem,
	pipelines []*gen.PipelineDefinition,
	sessions []chatagent.SessionSummary,
	apps []homelab.App,
) commandPaletteResults {
	q = strings.TrimSpace(q)
	out := commandPaletteResults{
		Pages:     []commandPaletteItem{},
		Pipelines: []commandPaletteItem{},
		Sessions:  []commandPaletteItem{},
		Homelab:   []commandPaletteItem{},
	}
	if q == "" {
		return out
	}
	needle := strings.ToLower(q)
	out.Pages = filterCommandPalettePages(needle, pages)
	out.Pipelines = filterCommandPalettePipelines(needle, pipelines)
	out.Sessions = filterCommandPaletteSessions(needle, sessions)
	out.Homelab = filterCommandPaletteHomelab(needle, apps)
	return out
}

func filterCommandPalettePages(needle string, pages []commandPaletteItem) []commandPaletteItem {
	out := []commandPaletteItem{}
	for _, p := range pages {
		if len(out) >= commandPaletteMaxPerGroup {
			break
		}
		if commandPaletteMatch(needle, p.Title, p.Subtitle) {
			out = append(out, p)
		}
	}
	return out
}

func filterCommandPalettePipelines(needle string, pipelines []*gen.PipelineDefinition) []commandPaletteItem {
	out := []commandPaletteItem{}
	for _, def := range pipelines {
		if def == nil {
			continue
		}
		if len(out) >= commandPaletteMaxPerGroup {
			break
		}
		if !commandPaletteMatch(needle, def.Name, def.Description) {
			continue
		}
		out = append(out, commandPaletteItem{
			ID:       "pipeline:" + def.Name,
			Title:    def.Name,
			Subtitle: def.Description,
			Href:     "/service/web/pipelines/" + url.PathEscape(def.Name),
			Group:    "pipelines",
		})
	}
	return out
}

func filterCommandPaletteSessions(needle string, sessions []chatagent.SessionSummary) []commandPaletteItem {
	out := []commandPaletteItem{}
	for _, s := range sessions {
		if len(out) >= commandPaletteMaxPerGroup {
			break
		}
		title := strings.TrimSpace(s.Title)
		if title == "" {
			title = s.SessionID
		}
		if !commandPaletteMatch(needle, title, s.SessionID) {
			continue
		}
		out = append(out, commandPaletteItem{
			ID:       "session:" + s.SessionID,
			Title:    title,
			Subtitle: "Session",
			Href:     "/service/web/agents/" + url.PathEscape(s.SessionID),
			Group:    "sessions",
		})
	}
	return out
}

func filterCommandPaletteHomelab(needle string, apps []homelab.App) []commandPaletteItem {
	out := []commandPaletteItem{}
	for _, app := range apps {
		if len(out) >= commandPaletteMaxPerGroup {
			break
		}
		if !commandPaletteMatch(needle, app.Name) {
			continue
		}
		out = append(out, commandPaletteItem{
			ID:       "homelab:" + app.Name,
			Title:    app.Name,
			Subtitle: "Homelab",
			Href:     "/service/web/homelab/" + url.PathEscape(app.Name),
			Group:    "homelab",
		})
	}
	return out
}

func commandPaletteMatch(needle string, fields ...string) bool {
	for _, f := range fields {
		if f == "" {
			continue
		}
		if strings.Contains(strings.ToLower(f), needle) {
			return true
		}
	}
	return false
}

// commandPaletteNavPages returns static jump targets aligned with the navbar.
func commandPaletteNavPages() []commandPaletteItem {
	nav := partials.CommandPaletteNavPages()
	out := make([]commandPaletteItem, 0, len(nav))
	for _, p := range nav {
		out = append(out, commandPaletteItem{
			ID:       p.ID,
			Title:    p.Title,
			Subtitle: p.Subtitle,
			Href:     p.Href,
			Group:    p.Group,
		})
	}
	return out
}

// recordCommandPaletteRecent prepends item onto existing recent visits, deduping by href
// and capping at commandPaletteRecentMax. Empty href is ignored.
func recordCommandPaletteRecent(existing []commandPaletteItem, item commandPaletteItem) []commandPaletteItem {
	if item.Href == "" {
		if existing == nil {
			return []commandPaletteItem{}
		}
		out := make([]commandPaletteItem, len(existing))
		copy(out, existing)
		return out
	}
	out := make([]commandPaletteItem, 0, len(existing)+1)
	out = append(out, item)
	for _, prev := range existing {
		if prev.Href == item.Href {
			continue
		}
		out = append(out, prev)
		if len(out) >= commandPaletteRecentMax {
			break
		}
	}
	return out
}
