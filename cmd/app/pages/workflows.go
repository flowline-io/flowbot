package pages

import (
	"fmt"

	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/components"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Workflows struct {
	app.Compo

	workflows  []admin.Workflow
	total      int64
	totalPages int

	page     int
	pageSize int

	search string
	status string

	loading bool
}

func (w *Workflows) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin/login")
		return
	}

	if w.page == 0 {
		w.page = 1
	}
	if w.pageSize == 0 {
		w.pageSize = 10
	}

	w.loadWorkflows(ctx)
}

func (w *Workflows) loadWorkflows(ctx app.Context) {
	w.loading = true
	token := state.Token(ctx)

	ctx.Async(func() {
		resp, err := api.ListWorkflows(token, w.page, w.pageSize, w.status, w.search)
		ctx.Dispatch(func(ctx app.Context) {
			w.loading = false
			if err != nil {
				components.ShowToast(ctx, "Failed to load workflows: "+err.Error(), "error")
				return
			}
			w.workflows = resp.Items
			w.total = resp.Total
			w.totalPages = resp.TotalPages
		})
	})
}

func (w *Workflows) Render() app.UI {
	return components.WithLayout(
		app.Div().Class("flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6").Body(
			app.Div().Body(
				app.H1().Class("text-3xl font-bold tracking-tight").Text("Workflows"),
				app.P().Class("text-base-content/50 mt-1").Text(
					fmt.Sprintf("%d workflows total", w.total),
				),
			),
			app.Div().Class("flex gap-2").Body(
				app.Button().
					Class("btn btn-primary btn-sm gap-2").
					OnClick(w.handleShowCreate).
					Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>`),
						app.Text("New Workflow"),
					),
			),
		),

		app.Div().Class("card bg-base-100 shadow-md").Body(
			app.Div().Class("card-body p-0").Body(
				app.Div().Class("px-6 pt-5 pb-3 flex flex-wrap gap-3").Body(
					app.Div().Class("relative max-w-xs").Body(
						app.Div().Class("absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none text-base-content/40").Body(
							app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>`),
						),
						app.Input().
							Type("text").
							Class("input input-bordered input-sm w-full pl-9").
							Placeholder("Search workflows...").
							Value(w.search).
							OnChange(w.handleSearch),
					),

					app.Select().
						Class("select select-bordered select-sm w-32").
						OnChange(w.handleStatusFilter).
						Body(
							app.Option().Value("").Selected(w.status == "").Text("All Status"),
							app.Option().Value("pending").Selected(w.status == "pending").Text("Pending"),
							app.Option().Value("running").Selected(w.status == "running").Text("Running"),
							app.Option().Value("completed").Selected(w.status == "completed").Text("Completed"),
							app.Option().Value("failed").Selected(w.status == "failed").Text("Failed"),
						),
				),

				app.Div().Class("overflow-x-auto").Body(
					app.If(w.loading, func() app.UI {
						return app.Div().Class("flex justify-center py-16").Body(
							app.Span().Class("loading loading-spinner loading-lg text-primary"),
						)
					}).Else(func() app.UI {
						if len(w.workflows) == 0 {
							return app.Div().Class("text-center py-16").Body(
								app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-4 text-base-content/30" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/></svg>`),
								app.P().Class("text-base-content/50").Text("No workflows found"),
							)
						}
						return app.Table().Class("table table-zebra w-full").Body(
							app.THead().Body(
								app.Tr().Class("text-base-content/60").Body(
									app.Th().Text("ID"),
									app.Th().Text("Name"),
									app.Th().Text("Description"),
									app.Th().Text("Status"),
									app.Th().Text("Created"),
									app.Th().Class("w-32").Text("Actions"),
								),
							),
							app.TBody().Body(
								w.renderRows()...,
							),
						)
					}),
				),

				app.Div().Class("px-6 py-4 border-t border-base-200").Body(
					w.renderPagination(),
				),
			),
		),

		w.renderModal(),
	)
}

func (w *Workflows) renderRows() []app.UI {
	rows := make([]app.UI, 0, len(w.workflows))
	for _, workflow := range w.workflows {
		wf := workflow
		rows = append(rows, app.Tr().Class("hover").Body(
			app.Td().Class("text-base-content/50 font-mono text-xs").Text(fmt.Sprintf("%d", wf.ID)),
			app.Td().Class("font-medium").Body(
				components.HighlightText(wf.Name, w.search),
			),
			app.Td().Class("text-base-content/60 text-sm").Body(
				components.HighlightText(w.truncateDescription(wf.Description), w.search),
			),
			app.Td().Body(w.statusBadge(wf.Status)),
			app.Td().Class("text-base-content/60 text-sm").Text(wf.CreatedAt.Format("2006-01-02 15:04")),
			app.Td().Body(
				app.Div().Class("flex gap-1").Body(
					app.If(wf.Status == admin.WorkflowPending, func() app.UI {
						return app.Button().
							Class("btn btn-ghost btn-xs gap-1 text-success").
							OnClick(w.handleRun(wf.ID)).
							Body(
								app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`),
								app.Text("Run"),
							)
					}),
					app.Button().
						Class("btn btn-ghost btn-xs text-error gap-1").
						OnClick(w.handleDelete(wf.ID)).
						Body(
							app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>`),
							app.Text("Delete"),
						),
				),
			),
		))
	}
	return rows
}

func (w *Workflows) statusBadge(status admin.WorkflowStatus) app.UI {
	badgeClass := "badge badge-sm gap-1 "
	switch status {
	case admin.WorkflowPending:
		badgeClass += "badge-warning"
	case admin.WorkflowRunning:
		badgeClass += "badge-info"
	case admin.WorkflowCompleted:
		badgeClass += "badge-success"
	case admin.WorkflowFailed:
		badgeClass += "badge-error"
	default:
		badgeClass += "badge-ghost"
	}
	return app.Span().Class(badgeClass).Text(string(status))
}

func (w *Workflows) truncateDescription(desc string) string {
	if len(desc) > 50 {
		return desc[:47] + "..."
	}
	return desc
}

func (w *Workflows) renderPagination() app.UI {
	if w.totalPages <= 1 {
		return app.Div()
	}

	buttons := make([]app.UI, 0, w.totalPages+2)

	buttons = append(buttons,
		app.Button().
			Class("join-item btn btn-sm").
			Disabled(w.page <= 1).
			OnClick(w.goToPage(w.page-1)).
			Body(app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>`)),
	)

	for i := 1; i <= w.totalPages; i++ {
		p := i
		cls := "join-item btn btn-sm"
		if p == w.page {
			cls += " btn-active"
		}
		buttons = append(buttons,
			app.Button().
				Class(cls).
				OnClick(w.goToPage(p)).
				Text(fmt.Sprintf("%d", p)),
		)
	}

	buttons = append(buttons,
		app.Button().
			Class("join-item btn btn-sm").
			Disabled(w.page >= w.totalPages).
			OnClick(w.goToPage(w.page+1)).
			Body(app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>`)),
	)

	return app.Div().Class("flex justify-center").Body(
		app.Div().Class("join").Body(buttons...),
	)
}

func (w *Workflows) renderModal() app.UI {
	return app.Div()
}

func (w *Workflows) handleSearch(ctx app.Context, e app.Event) {
	w.search = ctx.JSSrc().Get("value").String()
	w.page = 1
	w.loadWorkflows(ctx)
}

func (w *Workflows) handleStatusFilter(ctx app.Context, e app.Event) {
	w.status = ctx.JSSrc().Get("value").String()
	w.page = 1
	w.loadWorkflows(ctx)
}

func (w *Workflows) goToPage(page int) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		w.page = page
		w.loadWorkflows(ctx)
	}
}

func (w *Workflows) handleShowCreate(ctx app.Context, e app.Event) {
	components.ShowToast(ctx, "Workflow creation coming soon", "info")
}

func (w *Workflows) handleRun(id int64) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		token := state.Token(ctx)
		ctx.Async(func() {
			err := api.RunWorkflow(token, id)
			ctx.Dispatch(func(ctx app.Context) {
				if err != nil {
					components.ShowToast(ctx, "Failed to run workflow: "+err.Error(), "error")
					return
				}
				components.ShowToast(ctx, "Workflow started", "success")
				w.loadWorkflows(ctx)
			})
		})
	}
}

func (w *Workflows) handleDelete(id int64) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		token := state.Token(ctx)
		ctx.Async(func() {
			err := api.DeleteWorkflow(token, id)
			ctx.Dispatch(func(ctx app.Context) {
				if err != nil {
					components.ShowToast(ctx, "Failed to delete workflow: "+err.Error(), "error")
					return
				}
				components.ShowToast(ctx, "Workflow deleted", "success")
				w.loadWorkflows(ctx)
			})
		})
	}
}
