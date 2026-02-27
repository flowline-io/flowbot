package pages

import (
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/components"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// debounceDelay is the form-submit debounce delay.
const debounceDelay = 500 * time.Millisecond

// Settings is the system settings page component.
// The form supports debounced submission and loading-state display.
type Settings struct {
	app.Compo

	// Form fields
	siteName       string
	logoURL        string
	seoDescription string
	maxUploadSize  string // string input in MB

	// Status flags
	loading    bool
	saving     bool
	loadError  string
	debounceID int64 // debounce timer identifier
}

// OnNav checks login status and loads settings on page navigation.
func (s *Settings) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin/login")
		return
	}
	s.loadSettings(ctx)
}

// loadSettings fetches system settings from the backend.
func (s *Settings) loadSettings(ctx app.Context) {
	s.loading = true
	token := state.Token(ctx)

	ctx.Async(func() {
		settings, err := api.GetSettings(token)
		ctx.Dispatch(func(ctx app.Context) {
			s.loading = false
			if err != nil {
				s.loadError = "Failed to load settings: " + err.Error()
				components.ShowToast(ctx, s.loadError, "error")
				return
			}
			s.siteName = settings.SiteName
			s.logoURL = settings.LogoURL
			s.seoDescription = settings.SEODescription
			// Convert bytes to MB for display
			s.maxUploadSize = strconv.FormatInt(settings.MaxUploadSize/(1024*1024), 10)
		})
	})
}

// Render renders the system settings form.
func (s *Settings) Render() app.UI {
	return components.WithLayout(
		app.H1().Class("text-3xl font-bold mb-6").Text("System Settings"),

		app.If(s.loading, func() app.UI {
			return app.Div().Class("flex justify-center py-12").Body(
				app.Span().Class("loading loading-spinner loading-lg"),
			)
		}).Else(func() app.UI {
			return app.Div().Class("card bg-base-100 shadow-md max-w-2xl").Body(
				app.Div().Class("card-body").Body(
					// Site name
					s.formField("Site Name", "text", s.siteName, "Enter site name", func(ctx app.Context, e app.Event) {
						s.siteName = ctx.JSSrc().Get("value").String()
						s.debounceSave(ctx)
					}),
					// Logo URL
					s.formField("Logo URL", "url", s.logoURL, "https://example.com/logo.png", func(ctx app.Context, e app.Event) {
						s.logoURL = ctx.JSSrc().Get("value").String()
						s.debounceSave(ctx)
					}),
					// SEO description
					app.Div().Class("form-control mb-4").Body(
						app.Label().Class("label").Body(
							app.Span().Class("label-text font-medium").Text("SEO Description"),
						),
						app.Textarea().
							Class("textarea textarea-bordered h-24").
							Placeholder("Enter SEO description").
							Text(s.seoDescription).
							OnChange(func(ctx app.Context, e app.Event) {
								s.seoDescription = ctx.JSSrc().Get("value").String()
								s.debounceSave(ctx)
							}),
					),
					// Max upload size
					s.formField("Max Upload Size (MB)", "number", s.maxUploadSize, "10", func(ctx app.Context, e app.Event) {
						s.maxUploadSize = ctx.JSSrc().Get("value").String()
						s.debounceSave(ctx)
					}),

					// Submit button
					app.Div().Class("mt-6").Body(
						app.Button().
							Class("btn btn-primary").
							Disabled(s.saving).
							OnClick(s.handleSave).
							Body(
								app.If(s.saving, func() app.UI {
									return app.Span().Class("loading loading-spinner loading-sm mr-2")
								}),
								app.Text("Save Settings"),
							),
					),
				),
			)
		}),
	)
}

// formField renders a single form field.
func (s *Settings) formField(label, inputType, value, placeholder string, onChange app.EventHandler) app.UI {
	return app.Div().Class("form-control mb-4").Body(
		app.Label().Class("label").Body(
			app.Span().Class("label-text font-medium").Text(label),
		),
		app.Input().
			Type(inputType).
			Class("input input-bordered w-full").
			Value(value).
			Placeholder(placeholder).
			OnChange(onChange),
	)
}

// debounceSave debounces the save operation: delays execution after each input
// change, resetting the timer if more input arrives within the delay.
func (s *Settings) debounceSave(ctx app.Context) {
	s.debounceID++
	currentID := s.debounceID

	ctx.Async(func() {
		time.Sleep(debounceDelay)
		ctx.Dispatch(func(ctx app.Context) {
			// If debounceID has changed, new input arrived; skip this save
			if s.debounceID != currentID {
				return
			}
			s.doSave(ctx)
		})
	})
}

// handleSave handles the manual save button click.
func (s *Settings) handleSave(ctx app.Context, e app.Event) {
	s.doSave(ctx)
}

// doSave performs the actual save operation.
func (s *Settings) doSave(ctx app.Context) {
	if s.saving {
		return
	}
	s.saving = true

	// Convert MB to bytes
	mb, _ := strconv.ParseInt(s.maxUploadSize, 10, 64)
	if mb <= 0 {
		mb = 10
	}

	settings := admin.Settings{
		SiteName:       s.siteName,
		LogoURL:        s.logoURL,
		SEODescription: s.seoDescription,
		MaxUploadSize:  mb * 1024 * 1024,
	}

	token := state.Token(ctx)
	ctx.Async(func() {
		err := api.UpdateSettings(token, settings)
		ctx.Dispatch(func(ctx app.Context) {
			s.saving = false
			if err != nil {
				components.ShowToast(ctx, "Save failed: "+err.Error(), "error")
				return
			}
			components.ShowToast(ctx, "Settings saved", "success")
		})
	})
}
