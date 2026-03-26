package pages

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/components"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Logs struct {
	app.Compo

	logs       []admin.LogEntry
	total      int64
	totalPages int

	page     int
	pageSize int

	level  string
	source string
	search string

	autoRefresh bool
	loading     bool
}

func (l *Logs) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin/login")
		return
	}

	if l.page == 0 {
		l.page = 1
	}
	if l.pageSize == 1 {
		l.pageSize = 50
	}

	l.loadLogs(ctx)
}

func (l *Logs) loadLogs(ctx app.Context) {
	l.loading = true
	token := state.Token(ctx)

	ctx.Async(func() {
		resp, err := api.ListLogs(token, l.page, l.pageSize, l.level, l.source, l.search)
		ctx.Dispatch(func(ctx app.Context) {
			l.loading = false
			if err != nil {
				components.ShowToast(ctx, "Failed to load logs: "+err.Error(), "error")
				return
			}
			l.logs = resp.Items
			l.total = resp.Total
			l.totalPages = resp.TotalPages
		})
	})
}

func (l *Logs) Render() app.UI {
	return components.WithLayout(
		app.Div().Class("flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8").Body(
			app.Div().Body(
				app.H1().Class("text-3xl font-bold tracking-tight").Text("System Logs"),
				app.P().Class("text-base-content/50 mt-1").
					Text(fmt.Sprintf("%d log entries", l.total)),
			),
			app.Div().Class("flex gap-2").Body(
				app.If(l.autoRefresh, func() app.UI {
					return app.Button().
						Class("btn btn-primary btn-sm gap-2").
						OnClick(l.toggleAutoRefresh).
						Body(
							app.Span().Class("loading loading-spinner loading-xs"),
							app.Text("Auto-refresh ON"),
						)
				}).Else(func() app.UI {
					return app.Button().
						Class("btn btn-outline btn-sm gap-2").
						OnClick(l.toggleAutoRefresh).
						Body(
							app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.296 2H6M4 4v5h.582m15.296 2H6M4 4v5h.582m15.296 2H6m6 9l6-6-6-6"/></svg>`),
							app.Text("Auto-refresh OFF"),
						)
				}),
				app.Button().
					Class("btn btn-primary btn-sm gap-2").
					OnClick(l.handleRefresh).
					Disabled(l.loading).
					Body(
						app.If(l.loading, func() app.UI {
							return app.Span().Class("loading loading-spinner loading-xs")
						}),
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.296 2H6M4 4v5h.582m15.296 2H6"/></svg>`),
						app.Text("Refresh"),
					),
			),
		),

		app.Div().Class("card bg-base-100/80 backdrop-blur-sm shadow-xl border border-base-200/50 overflow-hidden").Body(
			app.Div().Class("card-body p-0").Body(
				app.Div().Class("px-6 pt-5 pb-4 border-b border-base-200/50").Body(
					app.Div().Class("flex flex-wrap gap-3 items-center").Body(
						app.Div().Class("relative max-w-md").Body(
							app.Div().Class("absolute inset-y-0 left-0 flex items-center pl-4 pointer-events-none text-base-content/40").Body(
								app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>`),
							),
							app.Input().
								Type("text").
								Class("input input-bordered input-md w-full pl-11 pr-4 bg-base-200/30 focus:bg-base-100 transition-colors duration-200").
								Placeholder("Search logs...").
								Value(l.search).
								OnChange(l.handleSearch),
						),

						app.Select().
							Class("select select-bordered select-md w-36 bg-base-200/30 focus:bg-base-100").
							OnChange(l.handleLevelFilter).
							Body(
								app.Option().Value("").Selected(l.level == "").Text("All Levels"),
								app.Option().Value("debug").Selected(l.level == "debug").Text("Debug"),
								app.Option().Value("info").Selected(l.level == "info").Text("Info"),
								app.Option().Value("warn").Selected(l.level == "warn").Text("Warn"),
								app.Option().Value("error").Selected(l.level == "error").Text("Error"),
							),

						app.Select().
							Class("select select-bordered select-md w-40 bg-base-200/30 focus:bg-base-100").
							OnChange(l.handleSourceFilter).
							Body(
								app.Option().Value("").Selected(l.source == "").Text("All Sources"),
								app.Option().Value("server").Selected(l.source == "server").Text("Server"),
								app.Option().Value("agent").Selected(l.source == "agent").Text("Agent"),
								app.Option().Value("workflow").Selected(l.source == "workflow").Text("Workflow"),
								app.Option().Value("platform").Selected(l.source == "platform").Text("Platform"),
							),
					),
				),

				app.Div().Class("overflow-x-auto max-h-[60vh] overflow-y-auto").Body(
					app.If(l.loading && len(l.logs) == 0, func() app.UI {
						return app.Div().Class("flex justify-center py-16").Body(
							app.Span().Class("loading loading-spinner loading-lg text-primary"),
						)
					}).Else(func() app.UI {
						return app.Div().Class("overflow-x-auto").Body(
							app.Table().Class("table").Body(
								app.THead().Body(
									app.Tr().Class("bg-base-200/30 sticky top-0").Body(
										app.Th().Class("w-20").Text("Level"),
										app.Th().Class("w-28").Text("Source"),
										app.Th().Text("Message"),
										app.Th().Class("w-44").Text("Time"),
									),
								),
								app.TBody().Body(
									l.renderRows()...,
								),
							),
						)
					}),
				),

				app.Div().Class("px-6 py-4 border-t border-base-200/50 bg-base-200/20").Body(
					l.renderPagination(),
				),
			),
		),
	)
}

func (l *Logs) renderRows() []app.UI {
	rows := make([]app.UI, 0, len(l.logs))
	for _, entry := range l.logs {
		e := entry
		rows = append(rows, app.Tr().Class("transition-colors duration-150 hover:bg-base-200/30").Body(
			app.Td().Body(l.levelBadge(e.Level)),
			app.Td().Class("text-base-content/60 text-sm font-mono").Text(e.Source),
			app.Td().Class("text-sm font-mono").Body(
				components.HighlightText(l.truncateMessage(e.Message, 200), l.search),
			),
			app.Td().Class("text-base-content/50 text-xs").Text(e.Timestamp),
		))
	}
	return rows
}

func (l *Logs) levelBadge(level admin.LogLevel) app.UI {
	badgeClass := "badge badge-sm gap-1 "
	switch level {
	case admin.LogLevelDebug:
		badgeClass += "badge-ghost"
	case admin.LogLevelInfo:
		badgeClass += "badge-info"
	case admin.LogLevelWarn:
		badgeClass += "badge-warning"
	case admin.LogLevelError:
		badgeClass += "badge-error"
	default:
		badgeClass += "badge-ghost"
	}
	return app.Span().Class(badgeClass).Text(string(level))
}

func (l *Logs) truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-3] + "..."
}

func (l *Logs) renderPagination() app.UI {
	if l.totalPages <= 1 {
		return app.Div().Class("flex justify-between items-center text-sm text-base-content/50").Body(
			app.Span().Text(fmt.Sprintf("Showing %d entries", len(l.logs))),
		)
	}

	buttons := make([]app.UI, 0, 5)

	buttons = append(buttons,
		app.Button().
			Class("join-item btn btn-sm").
			Disabled(l.page <= 1).
			OnClick(l.goToPage(l.page-1)).
			Body(app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>`)),
	)

	start := max(l.page-2, 1)
	end := min(l.page+2, l.totalPages)

	for i := start; i <= end; i++ {
		p := i
		cls := "join-item btn btn-sm"
		if p == l.page {
			cls += " btn-active"
		}
		buttons = append(buttons,
			app.Button().
				Class(cls).
				OnClick(l.goToPage(p)).
				Text(fmt.Sprintf("%d", p)),
		)
	}

	buttons = append(buttons,
		app.Button().
			Class("join-item btn btn-sm").
			Disabled(l.page >= l.totalPages).
			OnClick(l.goToPage(l.page+1)).
			Body(app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>`)),
	)

	return app.Div().Class("flex justify-between items-center").Body(
		app.Span().Class("text-sm text-base-content/50").Text(fmt.Sprintf("Showing %d of %d entries", len(l.logs), l.total)),
		app.Div().Class("join").Body(buttons...),
	)
}

func (l *Logs) handleSearch(ctx app.Context, e app.Event) {
	l.search = ctx.JSSrc().Get("value").String()
	l.page = 1
	l.loadLogs(ctx)
}

func (l *Logs) handleLevelFilter(ctx app.Context, e app.Event) {
	l.level = ctx.JSSrc().Get("value").String()
	l.page = 1
	l.loadLogs(ctx)
}

func (l *Logs) handleSourceFilter(ctx app.Context, e app.Event) {
	l.source = ctx.JSSrc().Get("value").String()
	l.page = 1
	l.loadLogs(ctx)
}

func (l *Logs) handleRefresh(ctx app.Context, e app.Event) {
	l.loadLogs(ctx)
}

func (l *Logs) goToPage(page int) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		l.page = page
		l.loadLogs(ctx)
	}
}

func (l *Logs) toggleAutoRefresh(ctx app.Context, e app.Event) {
	l.autoRefresh = !l.autoRefresh
	if l.autoRefresh {
		l.startAutoRefresh(ctx)
	}
}

func (l *Logs) startAutoRefresh(ctx app.Context) {
	ctx.Async(func() {
		for l.autoRefresh {
			time.Sleep(5 * time.Second)
			if !l.autoRefresh {
				return
			}
			ctx.Dispatch(func(ctx app.Context) {
				l.loadLogs(ctx)
			})
		}
	})
}
