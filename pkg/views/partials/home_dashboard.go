package partials

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
func HomeQuickLinks() []HomeQuickLink {
	return []HomeQuickLink{
		{Title: "Knowledge", Detail: "Markdown docs for agent retrieval", Href: "/service/web/agent-knowledge", TestID: "home-link-knowledge"},
		{Title: "Skills", Detail: "Reusable agent skill packages", Href: "/service/web/agent-skills", TestID: "home-link-skills"},
		{Title: "Notifications", Detail: "Channels, rules, and delivery", Href: "/service/web/notifications", TestID: "home-link-notifications"},
		{Title: "Health", Detail: "Infrastructure status", Href: "/service/web/healthz", TestID: "home-link-healthz"},
	}
}
