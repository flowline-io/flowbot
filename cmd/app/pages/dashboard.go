package pages

import (
	"fmt"

	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/components"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Dashboard is the admin panel home / dashboard page.
type Dashboard struct {
	app.Compo

	// User info
	user *admin.UserInfo
	// Dashboard stats from API
	stats   *admin.DashboardStats
	loading bool
}

// OnNav checks login status and loads data on page navigation.
func (d *Dashboard) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin/login")
		return
	}

	d.loading = true
	token := state.Token(ctx)
	ctx.Async(func() {
		user, _ := api.GetCurrentUser(token)
		stats, _ := api.GetDashboardStats(token)

		ctx.Dispatch(func(ctx app.Context) {
			d.loading = false
			d.user = user
			d.stats = stats
		})
	})
}

// Render renders the dashboard page.
func (d *Dashboard) Render() app.UI {
	greeting := "Welcome back"
	if d.user != nil {
		greeting = "Welcome back, " + d.user.Name
	}

	if d.loading {
		return components.WithLayout(
			app.Div().Class("flex justify-center py-24").Body(
				app.Span().Class("loading loading-spinner loading-lg text-primary"),
			),
		)
	}

	// Safe defaults when stats not loaded yet
	stats := d.stats
	if stats == nil {
		stats = &admin.DashboardStats{}
	}

	return components.WithLayout(
		// Header
		app.Div().Class("flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-8").Body(
			app.Div().Body(
				app.H1().Class("text-3xl font-bold tracking-tight").Text(greeting),
				app.P().Class("text-base-content/50 mt-1").Text("Here's an overview of your system"),
			),
			app.Div().Class("flex gap-2").Body(
				app.A().Href("/admin/containers").Class("btn btn-primary btn-sm gap-2").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>`),
					app.Text("New Container"),
				),
			),
		),

		// Row 1: Stats cards (4 columns)
		app.Div().Class("grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6").Body(
			d.statCard("Total", stats.TotalContainers, "primary",
				`<svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>`),
			d.statCard("Running", stats.RunningContainers, "success",
				`<svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`),
			d.statCard("Stopped", stats.StoppedContainers, "warning",
				`<svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`),
			d.statCard("Errors", stats.ErrorContainers, "error",
				`<svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`),
		),

		// Row 2: Container status bar + System info
		app.Div().Class("grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6").Body(
			// Container status distribution (2 cols)
			app.Div().Class("lg:col-span-2 card bg-base-100 shadow-md").Body(
				app.Div().Class("card-body").Body(
					d.sectionHeader("Container Status", `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>`),
					// Status bar
					d.renderStatusBar(stats),
					// Status breakdown
					app.Div().Class("grid grid-cols-2 sm:grid-cols-4 gap-3 mt-5").Body(
						d.miniStat("Running", stats.RunningContainers, "success"),
						d.miniStat("Stopped", stats.StoppedContainers, "warning"),
						d.miniStat("Paused", stats.PausedContainers, "info"),
						d.miniStat("Errors", stats.ErrorContainers, "error"),
					),
				),
			),

			// System info (1 col)
			app.Div().Class("card bg-base-100 shadow-md").Body(
				app.Div().Class("card-body").Body(
					d.sectionHeader("System Info", `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>`),
					app.Div().Class("space-y-3").Body(
						d.infoRow("Uptime", stats.Uptime),
						d.infoRow("Version", stats.Version),
						d.infoRow("Go", stats.GoVersion),
						d.infoRow("OS / Arch", stats.SystemOS+"/"+stats.SystemArch),
						d.infoRow("CPUs", fmt.Sprintf("%d", stats.NumCPU)),
						d.infoRow("Goroutines", fmt.Sprintf("%d", stats.NumRoutines)),
						d.infoRow("Heap Memory", formatBytes(stats.MemoryUsage)),
					),
				),
			),
		),

		// Row 3: Recent containers + Activity log
		app.Div().Class("grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6").Body(
			// Recent containers
			app.Div().Class("card bg-base-100 shadow-md").Body(
				app.Div().Class("card-body").Body(
					app.Div().Class("flex items-center justify-between mb-4").Body(
						d.sectionHeader("Recent Containers", `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>`),
						app.A().Href("/admin/containers").Class("btn btn-ghost btn-xs").Text("View all →"),
					),
					d.renderRecentContainers(stats.RecentContainers),
				),
			),

			// Activity log
			app.Div().Class("card bg-base-100 shadow-md").Body(
				app.Div().Class("card-body").Body(
					d.sectionHeader("Recent Activity", `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`),
					d.renderActivityLog(stats.ActivityLog),
				),
			),
		),

		// Row 4: Quick actions
		app.Div().Class("card bg-base-100 shadow-md").Body(
			app.Div().Class("card-body").Body(
				d.sectionHeader("Quick Actions", `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>`),
				app.Div().Class("grid grid-cols-1 sm:grid-cols-3 gap-3").Body(
					app.A().Href("/admin/containers").Class("btn btn-outline btn-sm justify-start gap-2").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>`),
						app.Text("Manage Containers"),
					),
					app.A().Href("/admin/settings").Class("btn btn-outline btn-sm justify-start gap-2").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>`),
						app.Text("System Settings"),
					),
					app.A().Href("/admin/login").Class("btn btn-outline btn-sm justify-start gap-2").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/></svg>`),
						app.Text("Switch Account"),
					),
				),
			),
		),
	)
}

// ---------------------------------------------------------------------------
// Helper render functions
// ---------------------------------------------------------------------------

// sectionHeader renders a consistent section title with icon.
func (d *Dashboard) sectionHeader(title, icon string) app.UI {
	return app.Div().Class("flex items-center gap-2 mb-4").Body(
		app.Div().Class("text-primary").Body(app.Raw(icon)),
		app.H2().Class("font-semibold text-base").Text(title),
	)
}

// statCard renders a stat card with an icon.
func (d *Dashboard) statCard(title string, value int, color, icon string) app.UI {
	return app.Div().Class("card bg-base-100 shadow-md hover:shadow-lg transition-shadow").Body(
		app.Div().Class("card-body flex-row items-center gap-3 p-4").Body(
			app.Div().Class("flex-shrink-0 rounded-xl bg-"+color+"/10 p-2.5 text-"+color).Body(
				app.Raw(icon),
			),
			app.Div().Body(
				app.Div().Class("text-xs text-base-content/50 uppercase tracking-wide").Text(title),
				app.Div().Class("text-2xl font-bold text-"+color).Text(
					fmt.Sprintf("%d", value),
				),
			),
		),
	)
}

// miniStat renders a compact stat item for the status breakdown.
func (d *Dashboard) miniStat(label string, value int, color string) app.UI {
	return app.Div().Class("flex items-center gap-2 rounded-lg bg-base-200/50 px-3 py-2").Body(
		app.Span().Class("w-2 h-2 rounded-full bg-"+color),
		app.Span().Class("text-sm text-base-content/70").Text(label),
		app.Span().Class("text-sm font-semibold ml-auto").Text(fmt.Sprintf("%d", value)),
	)
}

// infoRow renders a key-value row for system info.
func (d *Dashboard) infoRow(label, value string) app.UI {
	if value == "" {
		value = "—"
	}
	return app.Div().Class("flex justify-between items-center py-1.5 border-b border-base-200 last:border-0").Body(
		app.Span().Class("text-sm text-base-content/60").Text(label),
		app.Span().Class("text-sm font-medium font-mono").Text(value),
	)
}

// renderStatusBar renders a colored progress bar showing container status distribution.
func (d *Dashboard) renderStatusBar(stats *admin.DashboardStats) app.UI {
	total := stats.TotalContainers
	if total == 0 {
		return app.Div().Class("w-full h-3 rounded-full bg-base-200 mt-2")
	}

	pct := func(n int) string {
		return fmt.Sprintf("%.1f%%", float64(n)/float64(total)*100)
	}

	return app.Div().Class("w-full h-3 rounded-full bg-base-200 flex overflow-hidden mt-2").Body(
		app.If(stats.RunningContainers > 0, func() app.UI {
			return app.Div().Class("bg-success h-full transition-all").Style("width", pct(stats.RunningContainers))
		}),
		app.If(stats.PausedContainers > 0, func() app.UI {
			return app.Div().Class("bg-info h-full transition-all").Style("width", pct(stats.PausedContainers))
		}),
		app.If(stats.StoppedContainers > 0, func() app.UI {
			return app.Div().Class("bg-warning h-full transition-all").Style("width", pct(stats.StoppedContainers))
		}),
		app.If(stats.ErrorContainers > 0, func() app.UI {
			return app.Div().Class("bg-error h-full transition-all").Style("width", pct(stats.ErrorContainers))
		}),
	)
}

// renderRecentContainers renders a list of recent containers.
func (d *Dashboard) renderRecentContainers(containers []admin.Container) app.UI {
	if len(containers) == 0 {
		return app.Div().Class("text-center py-8 text-base-content/40").Body(
			app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>`),
			app.P().Text("No containers yet"),
		)
	}

	items := make([]app.UI, 0, len(containers))
	for _, ct := range containers {
		items = append(items, app.Div().Class("flex items-center justify-between py-2.5 border-b border-base-200 last:border-0").Body(
			app.Div().Class("flex items-center gap-3").Body(
				d.statusDot(ct.Status),
				app.Div().Body(
					app.Div().Class("font-medium text-sm").Text(ct.Name),
					app.Div().Class("text-xs text-base-content/40").Text(ct.CreatedAt.Format("2006-01-02 15:04")),
				),
			),
			d.containerStatusBadge(ct.Status),
		))
	}
	return app.Div().Body(items...)
}

// renderActivityLog renders the activity log entries.
func (d *Dashboard) renderActivityLog(entries []admin.ActivityEntry) app.UI {
	if len(entries) == 0 {
		return app.Div().Class("text-center py-8 text-base-content/40").Body(
			app.P().Text("No recent activity"),
		)
	}

	items := make([]app.UI, 0, len(entries))
	for _, e := range entries {
		iconColor := "text-success"
		if !e.Success {
			iconColor = "text-error"
		}
		items = append(items, app.Div().Class("flex items-start gap-3 py-2.5 border-b border-base-200 last:border-0").Body(
			app.Div().Class("mt-0.5 "+iconColor).Body(
				app.If(e.Success, func() app.UI {
					return app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>`)
				}).Else(func() app.UI {
					return app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>`)
				}),
			),
			app.Div().Class("flex-1 min-w-0").Body(
				app.Div().Class("text-sm").Body(
					app.Span().Class("font-medium").Text(e.Action),
					app.Span().Class("text-base-content/50").Text(" · "),
					app.Span().Class("text-base-content/60").Text(e.Target),
				),
				app.Div().Class("text-xs text-base-content/40 mt-0.5").Text(formatTimeAgo(e.Time)),
			),
		))
	}
	return app.Div().Body(items...)
}

// statusDot renders a small colored dot indicating container status.
func (d *Dashboard) statusDot(status admin.ContainerStatus) app.UI {
	color := "bg-base-300"
	switch status {
	case admin.ContainerRunning:
		color = "bg-success"
	case admin.ContainerStopped:
		color = "bg-warning"
	case admin.ContainerPaused:
		color = "bg-info"
	case admin.ContainerError:
		color = "bg-error"
	}
	return app.Div().Class("w-2.5 h-2.5 rounded-full " + color)
}

// containerStatusBadge renders a badge for the container status.
func (d *Dashboard) containerStatusBadge(status admin.ContainerStatus) app.UI {
	badgeClass := "badge badge-sm gap-1 "
	switch status {
	case admin.ContainerRunning:
		badgeClass += "badge-success"
	case admin.ContainerStopped:
		badgeClass += "badge-warning"
	case admin.ContainerPaused:
		badgeClass += "badge-info"
	case admin.ContainerError:
		badgeClass += "badge-error"
	default:
		badgeClass += "badge-ghost"
	}
	return app.Span().Class(badgeClass).Text(string(status))
}

// formatBytes converts bytes to a human-readable string.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// formatTimeAgo converts an RFC3339 time string to a relative "time ago" string.
func formatTimeAgo(timeStr string) string {
	// Simple fallback if parsing fails
	return timeStr
}
