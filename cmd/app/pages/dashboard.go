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
		// Greeting
		app.H1().Class("text-3xl font-bold mb-6").Text(greeting),

		// Stats cards
		app.Div().Class("grid grid-cols-1 md:grid-cols-3 gap-6 mb-8").Body(
			d.statCard("Total Containers", d.totalContainers, "primary"),
			d.statCard("Running", d.runningCount, "success"),
			d.statCard("Stopped", d.stoppedCount, "warning"),
		),

		// Quick actions
		app.Div().Class("card bg-base-100 shadow-md").Body(
			app.Div().Class("card-body").Body(
				app.H2().Class("card-title mb-4").Text("Quick Actions"),
				app.Div().Class("flex flex-wrap gap-3").Body(
					app.A().Href("/admin/containers").Class("btn btn-primary btn-sm").Text("Manage Containers"),
					app.A().Href("/admin/settings").Class("btn btn-outline btn-sm").Text("System Settings"),
				),
			),
		),
	)
}

// statCard renders a stat card.
func (d *Dashboard) statCard(title string, value int, color string) app.UI {
	return app.Div().Class("stat bg-base-100 shadow rounded-box").Body(
		app.Div().Class("stat-title").Text(title),
		app.Div().Class("stat-value text-"+color).Text(
			fmt.Sprintf("%d", value),
		),
	)
}
