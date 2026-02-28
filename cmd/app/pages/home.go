package pages

import (
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Home is the root "/" page that redirects based on authentication status.
type Home struct {
	app.Compo
}

// OnNav redirects to /admin if logged in, otherwise to /admin/login.
func (h *Home) OnNav(ctx app.Context) {
	if state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin")
	} else {
		ctx.Navigate("/admin/login")
	}
}

// Render returns an empty placeholder (the user is redirected in OnNav).
func (h *Home) Render() app.UI {
	return app.Div()
}
