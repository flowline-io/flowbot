package pages

import (
	"fmt"

	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/components"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Users struct {
	app.Compo

	users      []admin.User
	total      int64
	totalPages int

	page     int
	pageSize int

	search   string
	sortBy   string
	sortDesc bool

	selected map[int64]bool

	showModal bool
	editingID int64
	editName  string
	editEmail string
	editRole  string

	loading  bool
	deleting bool

	showConfirm    bool
	confirmTitle   string
	confirmMessage string
	confirmAction  func()
}

func (u *Users) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin/login")
		return
	}

	if u.page == 0 {
		u.page = 1
	}
	if u.pageSize == 0 {
		u.pageSize = 10
	}
	if u.selected == nil {
		u.selected = make(map[int64]bool)
	}

	u.loadData(ctx)
}

func (u *Users) loadData(ctx app.Context) {
	u.loading = true
	token := state.Token(ctx)

	ctx.Async(func() {
		resp, err := api.ListUsers(token, u.page, u.pageSize, u.search, u.sortBy, u.sortDesc)
		ctx.Dispatch(func(ctx app.Context) {
			u.loading = false
			if err != nil {
				components.ShowToast(ctx, "Failed to load users: "+err.Error(), "error")
				return
			}
			u.users = resp.Items
			u.total = resp.Total
			u.totalPages = resp.TotalPages
			u.selected = make(map[int64]bool)
		})
	})
}

func (u *Users) Render() app.UI {
	return components.WithLayout(
		app.Div().Class("flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8").Body(
			app.Div().Body(
				app.H1().Class("text-3xl font-bold tracking-tight").Text("Users"),
				app.P().Class("text-base-content/50 mt-1").
					Text(fmt.Sprintf("%d users total", u.total)),
			),
			app.Div().Class("flex gap-2").Body(
				app.If(u.selectedCount() > 0, func() app.UI {
					return app.Button().
						Class("btn btn-error btn-sm gap-2").
						Disabled(u.deleting).
						OnClick(u.showBatchDeleteConfirm).
						Body(
							app.If(u.deleting, func() app.UI {
								return app.Span().Class("loading loading-spinner loading-xs")
							}),
							app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>`),
							app.Text(fmt.Sprintf("Delete (%d)", u.selectedCount())),
						)
				}),
				app.Button().
					Class("btn btn-primary btn-sm gap-2").
					OnClick(u.handleShowCreate).
					Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>`),
						app.Text("New User"),
					),
			),
		),

		app.Div().Class("card bg-base-100/80 backdrop-blur-sm shadow-xl border border-base-200/50 overflow-hidden").Body(
			app.Div().Class("card-body p-0").Body(
				app.Div().Class("px-6 pt-5 pb-4 border-b border-base-200/50").Body(
					app.Div().Class("relative max-w-md").Body(
						app.Div().Class("absolute inset-y-0 left-0 flex items-center pl-4 pointer-events-none text-base-content/40").Body(
							app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>`),
						),
						app.Input().
							Type("text").
							Class("input input-bordered input-md w-full pl-11 pr-4 bg-base-200/30 focus:bg-base-100 transition-colors duration-200").
							Placeholder("Search users...").
							Value(u.search).
							OnChange(u.handleSearch),
					),
				),

				app.Div().Class("overflow-x-auto").Body(
					app.If(u.loading, func() app.UI {
						return app.Div().Class("flex justify-center py-16").Body(
							app.Span().Class("loading loading-spinner loading-lg text-primary"),
						)
					}).Else(func() app.UI {
						return app.Div().Class("overflow-x-auto").Body(
							app.Table().Class("table").Body(
								app.THead().Body(
									app.Tr().Class("bg-base-200/30").Body(
										app.Th().Class("w-12").Body(
											app.Input().
												Type("checkbox").
												Class("checkbox checkbox-sm checkbox-primary").
												Checked(u.isAllSelected()).
												OnChange(u.handleSelectAll),
										),
										u.sortableHeader("ID", "id"),
										u.sortableHeader("Name", "name"),
										u.sortableHeader("Email", "email"),
										u.sortableHeader("Role", "role"),
										u.sortableHeader("Status", "status"),
										u.sortableHeader("Created", "created_at"),
										app.Th().Class("w-16").Text("Actions"),
									),
								),
								app.TBody().Body(
									u.renderRows()...,
								),
							),
						)
					}),
				),

				app.Div().Class("px-6 py-4 border-t border-base-200/50 bg-base-200/20").Body(
					u.renderPagination(),
				),
			),
		),

		u.renderModal(),
		u.renderConfirmDialog(),
	)
}

func (u *Users) renderRows() []app.UI {
	rows := make([]app.UI, 0, len(u.users))
	for _, user := range u.users {
		usr := user
		rows = append(rows, app.Tr().Class("transition-colors duration-150 hover:bg-base-200/30").Body(
			app.Td().Body(
				app.Input().
					Type("checkbox").
					Class("checkbox checkbox-sm checkbox-primary").
					Checked(u.selected[usr.ID]).
					OnChange(u.toggleSelect(usr.ID)),
			),
			app.Td().Class("text-base-content/50 font-mono text-xs").Text(fmt.Sprintf("%d", usr.ID)),
			app.Td().Body(
				app.Div().Class("flex items-center gap-3").Body(
					app.Div().Class("avatar placeholder").Body(
						app.Div().Class("bg-gradient-to-br from-primary to-primary/70 text-white w-8 h-8 rounded-full flex items-center justify-center text-sm font-semibold").Body(
							app.Text(u.avatarInitial(usr.Name)),
						),
					),
					app.Span().Class("font-medium").Body(
						components.HighlightText(usr.Name, u.search),
					),
				),
			),
			app.Td().Class("text-base-content/60 text-sm").Body(
				components.HighlightText(usr.Email, u.search),
			),
			app.Td().Body(u.roleBadge(usr.Role)),
			app.Td().Body(u.statusBadge(usr.Status)),
			app.Td().Class("text-base-content/60 text-sm whitespace-nowrap").Text(usr.CreatedAt.Format("Jan 02 15:04")),
			app.Td().Body(
				app.Div().Class("flex gap-1").Body(
					app.Button().
						Class("btn btn-ghost btn-xs btn-circle hover:bg-primary/10 hover:text-primary transition-colors duration-150").
						Title("Edit").
						OnClick(u.handleEdit(usr)).
						Body(
							app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>`),
						),
					app.Button().
						Class("btn btn-ghost btn-xs btn-circle hover:bg-error/10 hover:text-error transition-colors duration-150").
						Title("Delete").
						OnClick(u.showDeleteConfirm(usr.ID)).
						Body(
							app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>`),
						),
				),
			),
		))
	}
	return rows
}

func (u *Users) sortableHeader(label, field string) app.UI {
	arrow := ""
	if u.sortBy == field {
		if u.sortDesc {
			arrow = " ▼"
		} else {
			arrow = " ▲"
		}
	}
	return app.Th().
		Class("cursor-pointer select-none hover:bg-base-200 transition-colors").
		OnClick(u.handleSort(field)).
		Text(label + arrow)
}

func (u *Users) roleBadge(role admin.UserRole) app.UI {
	badgeClass := "badge badge-sm gap-1 "
	switch role {
	case admin.RoleAdmin:
		badgeClass += "badge-primary"
	case admin.RoleUser:
		badgeClass += "badge-secondary"
	default:
		badgeClass += "badge-ghost"
	}
	return app.Span().Class(badgeClass).Text(string(role))
}

func (u *Users) statusBadge(status admin.UserStatus) app.UI {
	badgeClass := "badge badge-sm gap-1 "
	switch status {
	case admin.UserActive:
		badgeClass += "badge-success"
	case admin.UserInactive:
		badgeClass += "badge-warning"
	case admin.UserSuspended:
		badgeClass += "badge-error"
	default:
		badgeClass += "badge-ghost"
	}
	return app.Span().Class(badgeClass).Text(string(status))
}

func (u *Users) avatarInitial(name string) string {
	if name == "" {
		return "U"
	}
	return string([]rune(name)[0])
}

func (u *Users) renderPagination() app.UI {
	if u.totalPages <= 1 {
		return app.Div()
	}

	buttons := make([]app.UI, 0, u.totalPages+2)

	buttons = append(buttons,
		app.Button().
			Class("join-item btn btn-sm").
			Disabled(u.page <= 1).
			OnClick(u.goToPage(u.page-1)).
			Body(app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>`)),
	)

	for i := 1; i <= u.totalPages; i++ {
		p := i
		cls := "join-item btn btn-sm"
		if p == u.page {
			cls += " btn-active"
		}
		buttons = append(buttons,
			app.Button().
				Class(cls).
				OnClick(u.goToPage(p)).
				Text(fmt.Sprintf("%d", p)),
		)
	}

	buttons = append(buttons,
		app.Button().
			Class("join-item btn btn-sm").
			Disabled(u.page >= u.totalPages).
			OnClick(u.goToPage(u.page+1)).
			Body(app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>`)),
	)

	return app.Div().Class("flex justify-center").Body(
		app.Div().Class("join").Body(buttons...),
	)
}

func (u *Users) renderModal() app.UI {
	if !u.showModal {
		return app.Div()
	}

	title := "New User"
	if u.editingID > 0 {
		title = "Edit User"
	}

	return app.Div().Class("modal modal-open").Body(
		app.Div().Class("modal-box max-w-md").Body(
			app.Button().
				Class("btn btn-sm btn-circle btn-ghost absolute right-3 top-3").
				OnClick(u.handleCloseModal).
				Text("\u2715"),

			app.H3().Class("font-bold text-lg").Text(title),

			app.Div().Class("form-control mb-4").Body(
				app.Label().Class("label").Body(
					app.Span().Class("label-text font-medium").Text("Name"),
				),
				app.Input().
					Type("text").
					Class("input input-bordered w-full").
					Value(u.editName).
					Placeholder("Enter name").
					OnChange(func(ctx app.Context, e app.Event) {
						u.editName = ctx.JSSrc().Get("value").String()
					}),
			),

			app.Div().Class("form-control mb-4").Body(
				app.Label().Class("label").Body(
					app.Span().Class("label-text font-medium").Text("Email"),
				),
				app.Input().
					Type("email").
					Class("input input-bordered w-full").
					Value(u.editEmail).
					Placeholder("Enter email").
					OnChange(func(ctx app.Context, e app.Event) {
						u.editEmail = ctx.JSSrc().Get("value").String()
					}),
			),

			app.Div().Class("form-control mb-4").Body(
				app.Label().Class("label").Body(
					app.Span().Class("label-text font-medium").Text("Role"),
				),
				app.Select().
					Class("select select-bordered w-full").
					OnChange(func(ctx app.Context, e app.Event) {
						u.editRole = ctx.JSSrc().Get("value").String()
					}).
					Body(
						app.Option().Value("admin").Selected(u.editRole == "admin").Text("Admin"),
						app.Option().Value("user").Selected(u.editRole == "user").Text("User"),
						app.Option().Value("viewer").Selected(u.editRole == "viewer").Text("Viewer"),
					),
			),

			app.Div().Class("modal-action").Body(
				app.Button().
					Class("btn btn-ghost").
					OnClick(u.handleCloseModal).
					Text("Cancel"),
				app.Button().
					Class("btn btn-primary").
					OnClick(u.handleSaveUser).
					Text("Save"),
			),
		),
		app.Div().Class("modal-backdrop bg-black/40").OnClick(u.handleCloseModal),
	)
}

func (u *Users) renderConfirmDialog() app.UI {
	return &components.ConfirmDialog{
		Show:         u.showConfirm,
		Title:        u.confirmTitle,
		Message:      u.confirmMessage,
		ConfirmLabel: "Delete",
		ConfirmClass: "btn-error",
		OnConfirm: func() {
			if u.confirmAction != nil {
				u.confirmAction()
			}
			u.showConfirm = false
		},
		OnCancel: func() {
			u.showConfirm = false
			u.confirmAction = nil
		},
	}
}

func (u *Users) handleSearch(ctx app.Context, e app.Event) {
	u.search = ctx.JSSrc().Get("value").String()
	u.page = 1
	u.loadData(ctx)
}

func (u *Users) handleSort(field string) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		if u.sortBy == field {
			u.sortDesc = !u.sortDesc
		} else {
			u.sortBy = field
			u.sortDesc = false
		}
		u.page = 1
		u.loadData(ctx)
	}
}

func (u *Users) goToPage(page int) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		u.page = page
		u.loadData(ctx)
	}
}

func (u *Users) toggleSelect(id int64) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		if u.selected[id] {
			delete(u.selected, id)
		} else {
			u.selected[id] = true
		}
	}
}

func (u *Users) handleSelectAll(ctx app.Context, e app.Event) {
	if u.isAllSelected() {
		u.selected = make(map[int64]bool)
	} else {
		for _, usr := range u.users {
			u.selected[usr.ID] = true
		}
	}
}

func (u *Users) isAllSelected() bool {
	if len(u.users) == 0 {
		return false
	}
	for _, usr := range u.users {
		if !u.selected[usr.ID] {
			return false
		}
	}
	return true
}

func (u *Users) selectedCount() int {
	return len(u.selected)
}

func (u *Users) selectedIDs() []int64 {
	ids := make([]int64, 0, len(u.selected))
	for id := range u.selected {
		ids = append(ids, id)
	}
	return ids
}

func (u *Users) showBatchDeleteConfirm(ctx app.Context, e app.Event) {
	count := u.selectedCount()
	u.confirmTitle = "Delete Users"
	u.confirmMessage = fmt.Sprintf("Are you sure you want to delete %d users? This action cannot be undone.", count)
	u.confirmAction = func() {
		u.doBatchDelete(ctx)
	}
	u.showConfirm = true
}

func (u *Users) showDeleteConfirm(id int64) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		u.confirmTitle = "Delete User"
		u.confirmMessage = "Are you sure you want to delete this user? This action cannot be undone."
		u.confirmAction = func() {
			u.doDeleteUser(ctx, id)
		}
		u.showConfirm = true
	}
}

func (u *Users) doBatchDelete(ctx app.Context) {
	ids := u.selectedIDs()
	if len(ids) == 0 {
		return
	}

	u.deleting = true
	token := state.Token(ctx)

	ctx.Async(func() {
		var lastErr error
		for _, id := range ids {
			if err := api.DeleteUser(token, id); err != nil {
				lastErr = err
			}
		}
		ctx.Dispatch(func(ctx app.Context) {
			u.deleting = false
			if lastErr != nil {
				components.ShowToast(ctx, "Some deletions failed: "+lastErr.Error(), "error")
				return
			}
			components.ShowToast(ctx, fmt.Sprintf("Successfully deleted %d users", len(ids)), "success")
			u.loadData(ctx)
		})
	})
}

func (u *Users) doDeleteUser(ctx app.Context, id int64) {
	u.deleting = true
	token := state.Token(ctx)

	ctx.Async(func() {
		err := api.DeleteUser(token, id)
		ctx.Dispatch(func(ctx app.Context) {
			u.deleting = false
			if err != nil {
				components.ShowToast(ctx, "Delete failed: "+err.Error(), "error")
				return
			}
			components.ShowToast(ctx, "User deleted", "success")
			u.loadData(ctx)
		})
	})
}

func (u *Users) handleShowCreate(ctx app.Context, e app.Event) {
	u.editingID = 0
	u.editName = ""
	u.editEmail = ""
	u.editRole = "user"
	u.showModal = true
}

func (u *Users) handleEdit(usr admin.User) app.EventHandler {
	return func(ctx app.Context, e app.Event) {
		u.editingID = usr.ID
		u.editName = usr.Name
		u.editEmail = usr.Email
		u.editRole = string(usr.Role)
		u.showModal = true
	}
}

func (u *Users) handleCloseModal(ctx app.Context, e app.Event) {
	u.showModal = false
}

func (u *Users) handleSaveUser(ctx app.Context, e app.Event) {
	if u.editName == "" {
		components.ShowToast(ctx, "Name cannot be empty", "warning")
		return
	}
	if u.editEmail == "" {
		components.ShowToast(ctx, "Email cannot be empty", "warning")
		return
	}

	token := state.Token(ctx)
	u.showModal = false

	if u.editingID == 0 {
		req := admin.UserCreateRequest{
			Name:     u.editName,
			Email:    u.editEmail,
			Role:     admin.UserRole(u.editRole),
			Password: "temp1234",
		}
		ctx.Async(func() {
			_, err := api.CreateUser(token, req)
			ctx.Dispatch(func(ctx app.Context) {
				if err != nil {
					components.ShowToast(ctx, "Create failed: "+err.Error(), "error")
					return
				}
				components.ShowToast(ctx, "User created", "success")
				u.loadData(ctx)
			})
		})
	} else {
		req := admin.UserUpdateRequest{
			Name:  u.editName,
			Email: u.editEmail,
			Role:  admin.UserRole(u.editRole),
		}
		id := u.editingID
		ctx.Async(func() {
			_, err := api.UpdateUser(token, id, req)
			ctx.Dispatch(func(ctx app.Context) {
				if err != nil {
					components.ShowToast(ctx, "Update failed: "+err.Error(), "error")
					return
				}
				components.ShowToast(ctx, "User updated", "success")
				u.loadData(ctx)
			})
		})
	}
}
