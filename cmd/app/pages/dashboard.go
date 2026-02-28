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
	// Stats (demo)
	totalContainers int
	runningCount    int
	stoppedCount    int
}

// OnNav checks login status and loads data on page navigation.
func (d *Dashboard) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin/login")
		return
	}

	token := state.Token(ctx)
	ctx.Async(func() {
		// Load user info and container stats in parallel
		user, _ := api.GetCurrentUser(token)
		resp, _ := api.ListContainers(token, 1, 100, "", "", false)

		ctx.Dispatch(func(ctx app.Context) {
			d.user = user
			if resp != nil {
				d.totalContainers = int(resp.Total)
				for _, c := range resp.Items {
					switch c.Status {
					case admin.ContainerRunning:
						d.runningCount++
					case admin.ContainerStopped:
						d.stoppedCount++
					}
				}
			}
		})
	})
}

// Render renders the dashboard page.
func (d *Dashboard) Render() app.UI {
	greeting := "Welcome back"
	if d.user != nil {
		greeting = "Welcome back, " + d.user.Name
	}

	return components.WithLayout(
		// Greeting header section
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

		// Stats cards
		app.Div().Class("grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8").Body(
			d.statCard("Total Containers", d.totalContainers, "primary",
				`<svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>`),
			d.statCard("Running", d.runningCount, "success",
				`<svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5.636 18.364a9 9 0 010-12.728m12.728 0a9 9 0 010 12.728M9.172 15.828a4 4 0 010-5.656m5.656 0a4 4 0 010 5.656M12 12h.01"/></svg>`),
			d.statCard("Stopped", d.stoppedCount, "warning",
				`<svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`),
		),

		// Quick actions
		app.Div().Class("card bg-base-100 shadow-md").Body(
			app.Div().Class("card-body").Body(
				app.Div().Class("flex items-center gap-2 mb-4").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-base-content/60" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>`),
					app.H2().Class("card-title text-lg").Text("Quick Actions"),
				),
				app.Div().Class("grid grid-cols-1 sm:grid-cols-2 gap-3").Body(
					app.A().Href("/admin/containers").Class("btn btn-outline btn-sm justify-start gap-2").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>`),
						app.Text("Manage Containers"),
					),
					app.A().Href("/admin/settings").Class("btn btn-outline btn-sm justify-start gap-2").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>`),
						app.Text("System Settings"),
					),
				),
			),
		),
	)
}

// statCard renders a stat card with an icon.
func (d *Dashboard) statCard(title string, value int, color, icon string) app.UI {
	return app.Div().Class("card bg-base-100 shadow-md hover:shadow-lg transition-shadow").Body(
		app.Div().Class("card-body flex-row items-center gap-4 py-5").Body(
			app.Div().Class("flex-shrink-0 rounded-xl bg-"+color+"/10 p-3 text-"+color).Body(
				app.Raw(icon),
			),
			app.Div().Body(
				app.Div().Class("text-sm text-base-content/60").Text(title),
				app.Div().Class("text-2xl font-bold text-"+color).Text(
					fmt.Sprintf("%d", value),
				),
			),
		),
	)
}
