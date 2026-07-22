package partials

import "fmt"

// HomeDashboard is the view model for the authenticated home page.
type HomeDashboard struct {
	PipelineTotal    int64
	PipelineOK       int64
	PipelineFailed   int64
	Events24h        int64
	PostgresOK       bool
	RedisOK          bool
	UnhealthyCaps    int
	HubAppsTotal     int
	HubAppsRunning   int
	PendingApprovals int
	Checklist        []HomeChecklistItem
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
	Badge  int
}

// HomeQuickLinks returns the standard home shortcut set.
func HomeQuickLinks(pendingApprovals int) []HomeQuickLink {
	agents := HomeQuickLink{
		Title:  "Agents",
		Detail: "Chat and orchestrate",
		Href:   "/service/web/agents",
		TestID: "home-link-agents",
		Badge:  pendingApprovals,
	}
	if pendingApprovals > 0 {
		agents.Detail = fmt.Sprintf("%d awaiting approval", pendingApprovals)
	}
	return []HomeQuickLink{
		agents,
		{Title: "Pipelines", Detail: "Automate event flows", Href: "/service/web/pipelines", TestID: "home-link-pipelines"},
		{Title: "Hub", Detail: "Apps and capabilities", Href: "/service/web/hub", TestID: "home-link-hub"},
		{Title: "Health", Detail: "Infrastructure status", Href: "/service/web/healthz", TestID: "home-link-healthz"},
	}
}
