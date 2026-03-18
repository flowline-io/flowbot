package components

import (
	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Navbar is the top navigation bar component.
// Left side shows logo / site name; right side contains nav links, notification
// icon, and user avatar dropdown.
type Navbar struct {
	app.Compo

	user        *admin.UserInfo
	notifyCount int
	isDark      bool
}

func (n *Navbar) OnNav(ctx app.Context) {
	n.isDark = state.IsDarkMode(ctx)
	state.SetTheme(ctx, state.Theme(ctx))

	ctx.Handle("theme-changed", func(ctx app.Context, a app.Action) {
		ctx.Dispatch(func(ctx app.Context) {
			n.isDark = state.IsDarkMode(ctx)
		})
	})

	if !state.IsAuthenticated(ctx) {
		return
	}

	token := state.Token(ctx)
	ctx.Async(func() {
		user, err := api.GetCurrentUser(token)
		ctx.Dispatch(func(ctx app.Context) {
			if err != nil {
				return
			}
			n.user = user
		})
	})
}

// Render renders the navigation bar.
func (n *Navbar) Render() app.UI {
	return app.Div().Class("navbar bg-base-100/95 backdrop-blur-md border-b border-base-200/30 sticky top-0 z-50 shadow-sm").Body(
		// Left: Logo and site name
		app.Div().Class("flex-1").Body(
			app.A().Href("/admin").Class("btn btn-ghost text-xl normal-case gap-3 hover:bg-primary/5 transition-all duration-200").Body(
				app.Div().Class("w-9 h-9 rounded-xl bg-gradient-to-br from-primary to-primary/70 flex items-center justify-center shadow-lg shadow-primary/20").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>`),
				),
				app.Span().Class("font-bold text-gradient").Text("Flowbot"),
			),
		),

		// Center: Nav links (visible on desktop)
		app.Div().Class("hidden md:flex gap-1").Body(
			n.navLink("/admin", "Home"),
			n.navLink("/admin/users", "Users"),
			n.navLink("/admin/containers", "Containers"),
			n.navLink("/admin/workflows", "Workflows"),
			n.navLink("/admin/bots", "Bots"),
			n.navLink("/admin/logs", "Logs"),
			n.navLink("/admin/settings", "Settings"),
		),

		// Right: Notification + theme + user avatar
		app.Div().Class("flex-none gap-2").Body(
			// Theme toggle
			app.Button().Class("btn btn-ghost btn-circle hover:bg-primary/10 transition-all duration-200").OnClick(n.handleThemeToggle).Body(
				app.If(n.isDark, func() app.UI {
					return app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/></svg>`)
				}).Else(func() app.UI {
					return app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/></svg>`)
				}),
			),

			// Notification icon
			app.Button().Class("btn btn-ghost btn-circle hover:bg-primary/10 transition-all duration-200").Body(
				app.Div().Class("indicator").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/></svg>`),
					app.If(n.notifyCount > 0, func() app.UI {
						return app.Span().Class("badge badge-sm badge-primary indicator-item")
					}),
				),
			),

			// User avatar dropdown
			app.If(n.user != nil, func() app.UI {
				return app.Div().Class("dropdown dropdown-end").Body(
					app.Div().TabIndex(0).Attr("role", "button").Class("btn btn-ghost btn-circle avatar hover:bg-primary/5 transition-all duration-200").Body(
						app.Div().Class("w-10 rounded-full ring-2 ring-base-300 hover:ring-primary/40 transition-all duration-200").Body(
							app.If(n.user != nil && n.user.Avatar != "", func() app.UI {
								return app.Img().Src(n.user.Avatar).Alt(n.user.Name)
							}).Else(func() app.UI {
								return app.Div().Class("bg-gradient-to-br from-primary to-primary/70 text-white w-10 h-10 rounded-full flex items-center justify-center font-semibold text-sm shadow-md").Body(
									app.Text(n.avatarInitial()),
								)
							}),
						),
					),
					app.Ul().TabIndex(0).Class("dropdown-content menu bg-base-100/95 backdrop-blur-lg rounded-xl z-[1] w-56 p-2 shadow-xl border border-base-200/50 mt-3").Body(
						app.Li().Body(
							app.A().Class("rounded-lg hover:bg-primary/10 transition-colors duration-150").Text(n.displayName()),
						),
						app.Li().Body(
							app.A().OnClick(n.handleLogout).Class("rounded-lg hover:bg-error/10 hover:text-error transition-colors duration-150").Text("Logout"),
						),
					),
				)
			}),

			// Mobile hamburger menu
			app.Div().Class("dropdown dropdown-end md:hidden").Body(
				app.Div().TabIndex(0).Attr("role", "button").Class("btn btn-ghost hover:bg-primary/10 transition-colors duration-200").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>`),
				),
				app.Ul().TabIndex(0).Class("dropdown-content menu bg-base-100/95 backdrop-blur-lg rounded-xl z-[1] w-56 p-2 shadow-xl border border-base-200/50 mt-3").Body(
					app.Li().Body(app.A().Href("/admin").Class("rounded-lg hover:bg-primary/10 transition-colors duration-150").Text("Home")),
					app.Li().Body(app.A().Href("/admin/users").Class("rounded-lg hover:bg-primary/10 transition-colors duration-150").Text("Users")),
					app.Li().Body(app.A().Href("/admin/containers").Class("rounded-lg hover:bg-primary/10 transition-colors duration-150").Text("Containers")),
					app.Li().Body(app.A().Href("/admin/workflows").Class("rounded-lg hover:bg-primary/10 transition-colors duration-150").Text("Workflows")),
					app.Li().Body(app.A().Href("/admin/bots").Class("rounded-lg hover:bg-primary/10 transition-colors duration-150").Text("Bots")),
					app.Li().Body(app.A().Href("/admin/logs").Class("rounded-lg hover:bg-primary/10 transition-colors duration-150").Text("Logs")),
					app.Li().Body(app.A().Href("/admin/settings").Class("rounded-lg hover:bg-primary/10 transition-colors duration-150").Text("Settings")),
				),
			),
		),
	)
}

// navLink renders a single navigation link.
func (n *Navbar) navLink(href, label string) app.UI {
	return app.A().Href(href).Class("btn btn-ghost btn-sm normal-case font-medium rounded-lg hover:bg-primary/10 hover:text-primary transition-all duration-200").Text(label)
}

// avatarInitial returns the first character of the user's name (for default avatar).
func (n *Navbar) avatarInitial() string {
	if n.user == nil || n.user.Name == "" {
		return "U"
	}
	return string([]rune(n.user.Name)[0])
}

// displayName returns the display name for the user.
func (n *Navbar) displayName() string {
	if n.user == nil {
		return "User"
	}
	return n.user.Name
}

func (n *Navbar) handleLogout(ctx app.Context, e app.Event) {
	state.ClearToken(ctx)
	n.user = nil
	ctx.Navigate("/admin/login")
}

func (n *Navbar) handleThemeToggle(ctx app.Context, e app.Event) {
	n.isDark = !n.isDark
	if n.isDark {
		state.SetTheme(ctx, "dark")
	} else {
		state.SetTheme(ctx, "light")
	}
	ctx.NewAction("theme-changed")
}
