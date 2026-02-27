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

	// Current logged-in user info
	user *admin.UserInfo
	// Unread notification count (demo)
	notifyCount int
}

// OnNav loads user info on each navigation.
func (n *Navbar) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		return
	}

	token := state.Token(ctx)
	ctx.Async(func() {
		user, err := api.GetCurrentUser(token)
		ctx.Dispatch(func(ctx app.Context) {
			if err != nil {
				// Silently ignore errors to avoid blocking page rendering
				return
			}
			n.user = user
		})
	})
}

// Render renders the navigation bar.
func (n *Navbar) Render() app.UI {
	return app.Div().Class("navbar bg-base-100 shadow-md sticky top-0 z-50").Body(
		// Left: Logo and site name
		app.Div().Class("flex-1").Body(
			app.A().Href("/admin").Class("btn btn-ghost text-xl normal-case").Body(
				// Logo icon (SVG placeholder)
				app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>`),
				app.Text("Flowbot Admin"),
			),
		),

		// Center: Nav links (visible on desktop)
		app.Div().Class("hidden md:flex gap-1").Body(
			n.navLink("/admin", "Home"),
			n.navLink("/admin/containers", "Containers"),
			n.navLink("/admin/settings", "Settings"),
		),

		// Right: Notification + user avatar
		app.Div().Class("flex-none gap-2").Body(
			// Notification icon
			app.Button().Class("btn btn-ghost btn-circle").Body(
				app.Div().Class("indicator").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/></svg>`),
					app.If(n.notifyCount > 0, func() app.UI {
						return app.Span().Class("badge badge-sm badge-primary indicator-item").Text("●")
					}),
				),
			),

			// User avatar dropdown
			app.If(n.user != nil, func() app.UI {
				return app.Div().Class("dropdown dropdown-end").Body(
					app.Div().TabIndex(0).Attr("role", "button").Class("btn btn-ghost btn-circle avatar").Body(
						app.Div().Class("w-10 rounded-full").Body(
							app.If(n.user != nil && n.user.Avatar != "", func() app.UI {
								return app.Img().Src(n.user.Avatar).Alt(n.user.Name)
							}).Else(func() app.UI {
								// Default avatar placeholder
								return app.Div().Class("bg-neutral text-neutral-content w-10 h-10 rounded-full flex items-center justify-center").Body(
									app.Text(n.avatarInitial()),
								)
							}),
						),
					),
					app.Ul().TabIndex(0).Class("dropdown-content menu bg-base-100 rounded-box z-[1] w-52 p-2 shadow mt-3").Body(
						app.Li().Body(
							app.A().Text(n.displayName()),
						),
						app.Li().Body(
							app.A().OnClick(n.handleLogout).Text("Logout"),
						),
					),
				)
			}),

			// Mobile hamburger menu
			app.Div().Class("dropdown dropdown-end md:hidden").Body(
				app.Div().TabIndex(0).Attr("role", "button").Class("btn btn-ghost").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>`),
				),
				app.Ul().TabIndex(0).Class("dropdown-content menu bg-base-100 rounded-box z-[1] w-52 p-2 shadow mt-3").Body(
					app.Li().Body(app.A().Href("/admin").Text("Home")),
					app.Li().Body(app.A().Href("/admin/containers").Text("Containers")),
					app.Li().Body(app.A().Href("/admin/settings").Text("Settings")),
				),
			),
		),
	)
}

// navLink renders a single navigation link.
func (n *Navbar) navLink(href, label string) app.UI {
	return app.A().Href(href).Class("btn btn-ghost btn-sm normal-case").Text(label)
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

// handleLogout handles the logout action.
func (n *Navbar) handleLogout(ctx app.Context, e app.Event) {
	state.ClearToken(ctx)
	n.user = nil
	ctx.Navigate("/admin/login")
}
