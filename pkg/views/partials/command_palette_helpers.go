package partials

import (
	"context"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/i18n"
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
func CommandPaletteNavPages(ctx context.Context) []CommandPaletteNavPage {
	type page struct {
		titleKey, subtitleKey, href string
	}
	catalog := []page{
		{"nav.home", "nav.home_subtitle", "/service/web/home"},
		{"nav.agents", "nav.group.agent", "/service/web/agents"},
		{"nav.knowledge", "nav.group.agent", "/service/web/agent-knowledge"},
		{"nav.skills", "nav.group.agent", "/service/web/agent-skills"},
		{"nav.memory_facts", "nav.group.agent", "/service/web/agent-memory"},
		{"nav.session_summaries", "nav.group.agent", "/service/web/agent-session-summaries"},
		{"nav.scheduled_tasks", "nav.group.agent", "/service/web/agent-scheduled-tasks"},
		{"nav.subagents", "nav.group.agent", "/service/web/agent-subagents"},
		{"nav.sessions", "nav.group.agent", "/service/web/agent-sessions"},
		{"nav.permissions", "nav.group.agent", "/service/web/chatagent-settings"},
		{"nav.pipelines", "nav.group.automate", "/service/web/pipelines"},
		{"nav.workflows", "nav.group.automate", "/service/web/workflows"},
		{"nav.functions", "nav.group.automate", "/service/web/functions"},
		{"nav.events", "nav.group.automate", "/service/web/events"},
		{"nav.relations", "nav.group.automate", "/service/web/relations"},
		{"nav.apps", "nav.group.integrate", "/service/web/hub"},
		{"nav.registry", "nav.group.integrate", "/service/web/homelab"},
		{"nav.capabilities", "nav.group.integrate", "/service/web/capabilities"},
		{"nav.clips", "nav.group.integrate", "/service/web/clips"},
		{"nav.notifications", "nav.group.integrate", "/service/web/notifications"},
		{"nav.health", "nav.group.system", "/service/web/healthz"},
		{"nav.tokens", "nav.group.system", "/service/web/tokens"},
		{"nav.configs", "nav.group.system", "/service/web/configs"},
		{"nav.settings", "nav.group.system", "/service/web/settings"},
		{"nav.about", "nav.group.system", "/service/web/about"},
	}
	out := make([]CommandPaletteNavPage, 0, len(catalog))
	for _, p := range catalog {
		id := strings.TrimPrefix(p.href, "/service/web/")
		id = strings.ReplaceAll(id, "/", "-")
		out = append(out, CommandPaletteNavPage{
			ID:       "page:" + id,
			Title:    i18n.T(ctx, p.titleKey),
			Subtitle: i18n.T(ctx, p.subtitleKey),
			Href:     p.href,
			Group:    "pages",
		})
	}
	return out
}

// CommandPalettePagesJSON returns nav page jump targets as JSON for the command palette script.
func CommandPalettePagesJSON(ctx context.Context) string {
	b, err := sonic.MarshalString(CommandPaletteNavPages(ctx))
	if err != nil {
		return "[]"
	}
	return b
}
