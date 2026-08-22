package partials

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/i18n"
)

// HomeDashboard is the view model for the authenticated home page.
type HomeDashboard struct {
	ActiveSessions int
	PipelineTotal  int64
	PipelineOK     int64
	PipelineFailed int64
	WorkflowTotal  int64
	WorkflowOK     int64
	WorkflowFailed int64
	Events24h      int64
	Checklist      []HomeChecklistItem
}

// HomeChecklistItem is a setup step shown when the instance looks empty.
type HomeChecklistItem struct {
	Done   bool
	Title  string
	Detail string
	Href   string
	CTA    string
	TestID string
}

// HomeQuickLink is a shortcut card on the home dashboard.
type HomeQuickLink struct {
	Title  string
	Detail string
	Href   string
	TestID string
}

// HomeQuickLinks returns the standard home shortcut set.
func HomeQuickLinks(ctx context.Context) []HomeQuickLink {
	return []HomeQuickLink{
		{Title: i18n.T(ctx, "home.quick.knowledge.title"), Detail: i18n.T(ctx, "home.quick.knowledge.detail"), Href: "/service/web/agent-knowledge", TestID: "home-link-knowledge"},
		{Title: i18n.T(ctx, "home.quick.skills.title"), Detail: i18n.T(ctx, "home.quick.skills.detail"), Href: "/service/web/agent-skills", TestID: "home-link-skills"},
		{Title: i18n.T(ctx, "home.quick.notifications.title"), Detail: i18n.T(ctx, "home.quick.notifications.detail"), Href: "/service/web/notifications", TestID: "home-link-notifications"},
		{Title: i18n.T(ctx, "home.quick.health.title"), Detail: i18n.T(ctx, "home.quick.health.detail"), Href: "/service/web/healthz", TestID: "home-link-healthz"},
	}
}
