package pages

import (
	"fmt"

	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/components"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Containers is the container management CRUD page component.
// Features: paginated list, search filter, column sorting, checkbox bulk delete,
// create / edit / delete operations.
type Containers struct {
	app.Compo

	// List data
	containers []admin.Container
	total      int64
	totalPages int

	// Pagination
	page     int
	pageSize int

	// Search & sort
	search   string
	sortBy   string
	sortDesc bool

	// Bulk selection
	selected map[int64]bool

	// Create / edit modal
	showModal    bool
	editingID    int64  // 0 means creating new
	editName     string
	editStatus   string

	// Status flags
	loading  bool
	deleting bool
}

// OnNav initializes the page and loads data on navigation.
func (c *Containers) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin/login")
		return
	}

	// Initialize default values
	if c.page == 0 {
		c.page = 1
	}
	if c.pageSize == 0 {
		c.pageSize = 10
	}
	if c.selected == nil {
		c.selected = make(map[int64]bool)
	}

	c.loadData(ctx)
}

// loadData fetches the container list from the backend.
func (c *Containers) loadData(ctx app.Context) {
	c.loading = true
	token := state.Token(ctx)

	ctx.Async(func() {
		resp, err := api.ListContainers(token, c.page, c.pageSize, c.search, c.sortBy, c.sortDesc)
		ctx.Dispatch(func(ctx app.Context) {
			c.loading = false
			if err != nil {
				components.ShowToast(ctx, "Failed to load containers: "+err.Error(), "error")
				return
			}
			c.containers = resp.Items
			c.total = resp.Total
			c.totalPages = resp.TotalPages
			// Reset selection state
			c.selected = make(map[int64]bool)
		})
	})
}

// Render renders the container management page.
func (c *Containers) Render() app.UI {
	return components.WithLayout(
		// Page title and action bar
		app.Div().Class("flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6").Body(
			app.H1().Class("text-3xl font-bold").Text("Containers"),
			app.Div().Class("flex gap-2").Body(
				// Batch delete button
				app.If(c.selectedCount() > 0, func() app.UI {
					return app.Button().
						Class("btn btn-error btn-sm").
						Disabled(c.deleting).
						OnClick(c.handleBatchDelete).
						Body(
							app.If(c.deleting, func() app.UI {
								return app.Span().Class("loading loading-spinner loading-xs mr-1")
							}),
							app.Text(fmt.Sprintf("Delete (%d)", c.selectedCount())),
						)
				}),
				// Create button
				app.Button().
					Class("btn btn-primary btn-sm").
					OnClick(c.handleShowCreate).
				Text("+ New Container"),
			),
		),

		// Search bar
		app.Div().Class("mb-4").Body(
			app.Input().
				Type("text").
				Class("input input-bordered input-sm w-full max-w-xs").
				Placeholder("Search containers...").
				Value(c.search).
				OnChange(c.handleSearch),
		),

		// Data table
		app.Div().Class("overflow-x-auto").Body(
			app.If(c.loading, func() app.UI {
				return app.Div().Class("flex justify-center py-12").Body(
					app.Span().Class("loading loading-spinner loading-lg"),
				)
			}).Else(func() app.UI {
				return app.Table().Class("table table-zebra w-full").Body(
				// Table header
					app.THead().Body(
						app.Tr().Body(
				// Select-all checkbox
							app.Th().Body(
								app.Input().
									Type("checkbox").
									Class("checkbox checkbox-sm").
									Checked(c.isAllSelected()).
									OnChange(c.handleSelectAll),
							),
							c.sortableHeader("ID", "id"),
						c.sortableHeader("Name", "name"),
						c.sortableHeader("Status", "status"),
						c.sortableHeader("Created At", "created_at"),
						app.Th().Text("Actions"),
						),
					),
					// Table body
					app.TBody().Body(
						c.renderRows()...,
					),
				)
			}),
		),

		// Pagination
		c.renderPagination(),

		// Create / edit modal
		c.renderModal(),
	)
}

// renderRows renders table rows.
func (c *Containers) renderRows() []app.UI {
	rows := make([]app.UI, 0, len(c.containers))
	for _, container := range c.containers {
		ct := container // capture for closure
		rows = append(rows, app.Tr().Body(
			// Checkbox
			app.Td().Body(
				app.Input().
					Type("checkbox").
					Class("checkbox checkbox-sm").
					Checked(c.selected[ct.ID]).
					OnChange(c.toggleSelect(ct.ID)),
			),
			// ID
			app.Td().Text(fmt.Sprintf("%d", ct.ID)),
			// Name
			app.Td().Class("font-medium").Text(ct.Name),
			// Status (Badge)
			app.Td().Body(c.statusBadge(ct.Status)),
			// Created at
			app.Td().Text(ct.CreatedAt.Format("2006-01-02 15:04:05")),
			// Action buttons
			app.Td().Body(
				app.Div().Class("flex gap-1").Body(
					app.Button().
						Class("btn btn-ghost btn-xs").
						OnClick(c.handleEdit(ct)).
					Text("Edit"),
					app.Button().
						Class("btn btn-ghost btn-xs text-error").
						OnClick(c.handleDelete(ct.ID)).
					Text("Delete"),
				),
			),
		))
	}
	return rows
}

// sortableHeader renders a sortable column header.
func (c *Containers) sortableHeader(label, field string) app.UI {
	arrow := ""
	if c.sortBy == field {
		if c.sortDesc {
			arrow = " ▼"
		} else {
			arrow = " ▲"
		}
	}
	return app.Th().
		Class("cursor-pointer select-none hover:bg-base-200").
		OnClick(c.handleSort(field)).
		Text(label + arrow)
}

// statusBadge renders a DaisyUI Badge based on the container status.
func (c *Containers) statusBadge(status admin.ContainerStatus) app.UI {
	badgeClass := "badge badge-sm "
	switch status {
	case admin.ContainerRunning:
		badgeClass += "badge-success"
	case admin.ContainerStopped:
		badgeClass += "badge-error"
	case admin.ContainerPaused:
		badgeClass += "badge-warning"
	default:
		badgeClass += "badge-ghost"
	}
	return app.Span().Class(badgeClass).Text(string(status))
}

// renderPagination renders the pagination controls.
func (c *Containers) renderPagination() app.UI {
	if c.totalPages <= 1 {
		return app.Div()
	}

	buttons := make([]app.UI, 0, c.totalPages+2)

	// Previous page
	buttons = append(buttons,
		app.Button().
			Class("join-item btn btn-sm").
			Disabled(c.page <= 1).
			OnClick(c.goToPage(c.page-1)).
			Text("«"),
	)

	// Page number buttons
	for i := 1; i <= c.totalPages; i++ {
		p := i
		cls := "join-item btn btn-sm"
		if p == c.page {
			cls += " btn-active"
		}
		buttons = append(buttons,
			app.Button().
				Class(cls).
				OnClick(c.goToPage(p)).
				Text(fmt.Sprintf("%d", p)),
		)
	}

	// Next page
	buttons = append(buttons,
		app.Button().
			Class("join-item btn btn-sm").
			Disabled(c.page >= c.totalPages).
			OnClick(c.goToPage(c.page+1)).
			Text("»"),
	)

	return app.Div().Class("flex justify-center mt-6").Body(
		app.Div().Class("join").Body(buttons...),
	)
}

// renderModal renders the create/edit modal dialog.
func (c *Containers) renderModal() app.UI {
	if !c.showModal {
		return app.Div()
	}

	title := "New Container"
	if c.editingID > 0 {
		title = "Edit Container"
	}

	return app.Div().Class("modal modal-open").Body(
		app.Div().Class("modal-box").Body(
			app.H3().Class("font-bold text-lg mb-4").Text(title),

			// Container name
			app.Div().Class("form-control mb-3").Body(
				app.Label().Class("label").Body(
					app.Span().Class("label-text").Text("Container Name"),
				),
				app.Input().
					Type("text").
					Class("input input-bordered w-full").
					Value(c.editName).
					Placeholder("Enter container name").
					OnChange(func(ctx app.Context, e app.Event) {
						c.editName = ctx.JSSrc().Get("value").String()
					}),
			),

			// Container status
			app.Div().Class("form-control mb-4").Body(
				app.Label().Class("label").Body(
					app.Span().Class("label-text").Text("Status"),
				),
				app.Select().
					Class("select select-bordered w-full").
					OnChange(func(ctx app.Context, e app.Event) {
						c.editStatus = ctx.JSSrc().Get("value").String()
					}).
					Body(
						app.Option().Value("running").Selected(c.editStatus == "running").Text("Running"),
						app.Option().Value("stopped").Selected(c.editStatus == "stopped").Text("Stopped"),
						app.Option().Value("paused").Selected(c.editStatus == "paused").Text("Paused"),
					),
			),

			// Action buttons
			app.Div().Class("modal-action").Body(
				app.Button().
					Class("btn btn-ghost").
					OnClick(c.handleCloseModal).
					Text("Cancel"),
				app.Button().
					Class("btn btn-primary").
					OnClick(c.handleSaveContainer).
					Text("Save"),
			),
		),
		// Click backdrop to close
		app.Div().Class("modal-backdrop").OnClick(c.handleCloseModal),
	)
}

// ---------------------------------------------------------------------------
// Event handlers
// ---------------------------------------------------------------------------

// handleSearch handles search input.
func (c *Containers) handleSearch(ctx app.Context, e app.Event) {
	c.search = ctx.JSSrc().Get("value").String()
	c.page = 1
	c.loadData(ctx)
}

// handleSort handles column sorting.
func (c *Containers) handleSort(field string) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		if c.sortBy == field {
			c.sortDesc = !c.sortDesc
		} else {
			c.sortBy = field
			c.sortDesc = false
		}
		c.page = 1
		c.loadData(ctx)
	}
}

// goToPage navigates to a specific page.
func (c *Containers) goToPage(page int) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		c.page = page
		c.loadData(ctx)
	}
}

// toggleSelect toggles the selection state of a single row.
func (c *Containers) toggleSelect(id int64) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		if c.selected[id] {
			delete(c.selected, id)
		} else {
			c.selected[id] = true
		}
	}
}

// handleSelectAll toggles select-all / deselect-all.
func (c *Containers) handleSelectAll(ctx app.Context, e app.Event) {
	if c.isAllSelected() {
		c.selected = make(map[int64]bool)
	} else {
		for _, ct := range c.containers {
			c.selected[ct.ID] = true
		}
	}
}

// isAllSelected checks whether all rows are selected.
func (c *Containers) isAllSelected() bool {
	if len(c.containers) == 0 {
		return false
	}
	for _, ct := range c.containers {
		if !c.selected[ct.ID] {
			return false
		}
	}
	return true
}

// selectedCount returns the number of selected items.
func (c *Containers) selectedCount() int {
	return len(c.selected)
}

// selectedIDs returns the list of selected IDs.
func (c *Containers) selectedIDs() []int64 {
	ids := make([]int64, 0, len(c.selected))
	for id := range c.selected {
		ids = append(ids, id)
	}
	return ids
}

// handleBatchDelete performs batch deletion.
func (c *Containers) handleBatchDelete(ctx app.Context, e app.Event) {
	ids := c.selectedIDs()
	if len(ids) == 0 {
		return
	}

	c.deleting = true
	token := state.Token(ctx)

	ctx.Async(func() {
		err := api.BatchDeleteContainers(token, ids)
		ctx.Dispatch(func(ctx app.Context) {
			c.deleting = false
			if err != nil {
				components.ShowToast(ctx, "Batch delete failed: "+err.Error(), "error")
				return
			}
			components.ShowToast(ctx, fmt.Sprintf("Successfully deleted %d containers", len(ids)), "success")
			c.loadData(ctx)
		})
	})
}

// handleDelete deletes a single container.
func (c *Containers) handleDelete(id int64) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		token := state.Token(ctx)
		ctx.Async(func() {
			err := api.DeleteContainer(token, id)
			ctx.Dispatch(func(ctx app.Context) {
				if err != nil {
					components.ShowToast(ctx, "Delete failed: "+err.Error(), "error")
					return
				}
				components.ShowToast(ctx, "Container deleted", "success")
				c.loadData(ctx)
			})
		})
	}
}

// handleShowCreate opens the create modal.
func (c *Containers) handleShowCreate(ctx app.Context, e app.Event) {
	c.editingID = 0
	c.editName = ""
	c.editStatus = "running"
	c.showModal = true
}

// handleEdit opens the edit modal.
func (c *Containers) handleEdit(ct admin.Container) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		c.editingID = ct.ID
		c.editName = ct.Name
		c.editStatus = string(ct.Status)
		c.showModal = true
	}
}

// handleCloseModal closes the modal.
func (c *Containers) handleCloseModal(ctx app.Context, e app.Event) {
	c.showModal = false
}

// handleSaveContainer saves a container (create or edit).
func (c *Containers) handleSaveContainer(ctx app.Context, e app.Event) {
	if c.editName == "" {
		components.ShowToast(ctx, "Container name cannot be empty", "warning")
		return
	}

	token := state.Token(ctx)
	c.showModal = false

	if c.editingID == 0 {
		// Create new
		req := admin.ContainerCreateRequest{
			Name:   c.editName,
			Status: admin.ContainerStatus(c.editStatus),
		}
		ctx.Async(func() {
			_, err := api.CreateContainer(token, req)
			ctx.Dispatch(func(ctx app.Context) {
				if err != nil {
					components.ShowToast(ctx, "Create failed: "+err.Error(), "error")
					return
				}
				components.ShowToast(ctx, "Container created", "success")
				c.loadData(ctx)
			})
		})
	} else {
		// Edit existing
		req := admin.ContainerUpdateRequest{
			Name:   c.editName,
			Status: admin.ContainerStatus(c.editStatus),
		}
		id := c.editingID
		ctx.Async(func() {
			_, err := api.UpdateContainer(token, id, req)
			ctx.Dispatch(func(ctx app.Context) {
				if err != nil {
					components.ShowToast(ctx, "Update failed: "+err.Error(), "error")
					return
				}
				components.ShowToast(ctx, "Container updated", "success")
				c.loadData(ctx)
			})
		})
	}
}
