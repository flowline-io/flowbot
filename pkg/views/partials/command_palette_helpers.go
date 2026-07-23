package partials

import (
	"strings"

	"github.com/bytedance/sonic"
)

// CommandPaletteNavPage is one static jump target for the global command palette.
type CommandPaletteNavPage struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Href     string `json:"href"`
	Group    string `json:"group"`
}

// CommandPaletteNavPages returns static jump targets aligned with the navbar.
func CommandPaletteNavPages() []CommandPaletteNavPage {
	type page struct {
		title, subtitle, href string
	}
	catalog := []page{
		{"Home", "Dashboard", "/service/web/home"},
		{"Agents", "Agent", "/service/web/agents"},
		{"Skills", "Agent", "/service/web/agent-skills"},
		{"Knowledge", "Agent", "/service/web/agent-knowledge"},
		{"Memory Facts", "Agent", "/service/web/agent-memory"},
		{"Session Summaries", "Agent", "/service/web/agent-session-summaries"},
		{"Subagents", "Agent", "/service/web/agent-subagents"},
		{"Scheduled Tasks", "Agent", "/service/web/agent-scheduled-tasks"},
		{"Sessions", "Agent", "/service/web/agent-sessions"},
		{"Permissions", "Agent", "/service/web/chatagent-permissions"},
		{"Pipelines", "Automate", "/service/web/pipelines"},
		{"Workflows", "Automate", "/service/web/workflows"},
		{"Events", "Automate", "/service/web/events"},
		{"Relations", "Automate", "/service/web/relations"},
		{"Apps", "Integrate", "/service/web/hub"},
		{"Registry", "Integrate", "/service/web/homelab"},
		{"Capabilities", "Integrate", "/service/web/capabilities"},
		{"Clips", "Integrate", "/service/web/clips"},
		{"Notifications", "Integrate", "/service/web/notifications"},
		{"Health", "System", "/service/web/healthz"},
		{"Tokens", "System", "/service/web/tokens"},
		{"Configs", "System", "/service/web/configs"},
	}
	out := make([]CommandPaletteNavPage, 0, len(catalog))
	for _, p := range catalog {
		id := strings.TrimPrefix(p.href, "/service/web/")
		id = strings.ReplaceAll(id, "/", "-")
		out = append(out, CommandPaletteNavPage{
			ID:       "page:" + id,
			Title:    p.title,
			Subtitle: p.subtitle,
			Href:     p.href,
			Group:    "pages",
		})
	}
	return out
}

// CommandPalettePagesJSON returns nav page jump targets as JSON for the command palette script.
func CommandPalettePagesJSON() string {
	b, err := sonic.MarshalString(CommandPaletteNavPages())
	if err != nil {
		return "[]"
	}
	return b
}
