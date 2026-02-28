// Package pages contains all page components for the Admin panel.
package pages

import (
	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/components"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Login is the Slack OAuth login page component.
type Login struct {
	app.Compo

	// loading indicates whether a login request is in progress.
	loading bool
	// errMsg holds the login error message, if any.
	errMsg string
}

// OnNav checks the URL for a code or error parameter from the OAuth callback.
func (l *Login) OnNav(ctx app.Context) {
	// If already logged in, navigate directly to the dashboard
	if state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin")
		return
	}

	// Check whether the URL carries an error from the OAuth callback
	if errMsg := ctx.Page().URL().Query().Get("error"); errMsg != "" {
		l.errMsg = errMsg
	}

	// Check whether the URL carries a one-time exchange code from the OAuth callback
	code := ctx.Page().URL().Query().Get("code")
	if code != "" {
		l.loading = true
		ctx.Async(func() {
			token, err := api.ExchangeCode(code)
			ctx.Dispatch(func(ctx app.Context) {
				l.loading = false
				if err != nil {
					l.errMsg = "Login failed: " + err.Error()
					return
				}
				state.SetToken(ctx, token)
				components.ShowToast(ctx, "Login successful", "success")
				ctx.Navigate("/admin")
			})
		})
	}
}

// Render renders the login page.
func (l *Login) Render() app.UI {
	return components.WithMinimalLayout(
		app.Div().Class("hero min-h-[90vh]").Body(
			app.Div().Class("hero-content flex-col w-full max-w-lg").Body(
				// Logo & branding
				app.Div().Class("text-center mb-2").Body(
					app.Div().Class("inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 mb-4").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>`),
					),
					app.H1().Class("text-3xl font-bold").Text("Flowbot Admin"),
					app.P().Class("text-base-content/50 mt-1").Text("Sign in to manage your workspace"),
				),

				// Card
				app.Div().Class("card w-full bg-base-100 shadow-2xl").Body(
					app.Div().Class("card-body gap-5 px-8 py-8").Body(
						// Error message
						app.If(l.errMsg != "", func() app.UI {
							return app.Div().Class("alert alert-error shadow-sm").Body(
								app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`),
								app.Span().Text(l.errMsg),
							)
						}),

						// Login with Slack button
						app.Button().
							Class("btn btn-primary btn-block gap-2 h-12 text-base").
							Disabled(l.loading).
							OnClick(l.handleSlackLogin).
							Body(
								app.If(l.loading, func() app.UI {
									return app.Span().Class("loading loading-spinner loading-sm")
								}),
								app.Raw(`<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zM6.313 15.165a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zM8.834 6.313a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312zM18.956 8.834a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zM17.688 8.834a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312zM15.165 18.956a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zM15.165 17.688a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.527 2.527 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z"/></svg>`),
								app.Text("Login with Slack"),
							),

						// Divider
						app.Div().Class("divider text-base-content/40 text-xs my-1").Text("OR"),

						// Dev-mode quick login button
						app.Button().
							Class("btn btn-outline btn-block btn-sm gap-2").
							Disabled(l.loading).
							OnClick(l.handleDevLogin).
							Body(
								app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"/></svg>`),
								app.Text("Dev Quick Login"),
							),
					),
				),

				// Footer text
				app.P().Class("text-center text-xs text-base-content/40 mt-4").
					Text("Powered by Flowbot"),
			),
		),
	)
}

// handleSlackLogin handles the Slack OAuth login button click.
func (l *Login) handleSlackLogin(ctx app.Context, e app.Event) {
	l.loading = true
	l.errMsg = ""

	token := state.Token(ctx)
	ctx.Async(func() {
		oauthURL, err := api.GetSlackOAuthURL(token)
		ctx.Dispatch(func(ctx app.Context) {
			l.loading = false
			if err != nil {
				l.errMsg = "Failed to get Slack OAuth URL: " + err.Error()
				return
			}
			// Navigate to the Slack authorization page (external link)
			ctx.Navigate(oauthURL)
		})
	})
}

// handleDevLogin handles the dev-mode quick login.
func (l *Login) handleDevLogin(ctx app.Context, e app.Event) {
	l.loading = true
	l.errMsg = ""

	ctx.Async(func() {
		newToken, err := api.DevLogin("")
		ctx.Dispatch(func(ctx app.Context) {
			l.loading = false
			if err != nil {
				l.errMsg = "Login failed: " + err.Error()
				return
			}
			state.SetToken(ctx, newToken)
			components.ShowToast(ctx, "Login successful", "success")
			ctx.Navigate("/admin")
		})
	})
}
