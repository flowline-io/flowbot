// Package components provides reusable UI components for the Admin frontend.
package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// WithLayout wraps page content in a unified layout skeleton: Navbar + main content area + Toast + Loading.
func WithLayout(content ...app.UI) app.UI {
	allContent := make([]app.UI, 0, len(content)+1)
	allContent = append(allContent, &Breadcrumb{})
	allContent = append(allContent, content...)

	return app.Div().Class("min-h-screen bg-gradient-to-br from-base-100 via-base-200/30 to-base-100 flex flex-col").Body(
		&Navbar{},

		app.Main().Class("flex-1 container mx-auto px-4 sm:px-6 py-8 max-w-7xl").Body(
			allContent...,
		),

		app.Footer().Class("footer footer-center py-6 text-base-content/30 text-sm mt-auto border-t border-base-200/30").Body(
			app.P().Text("Powered by Flowbot"),
		),

		&Toast{},
		&LoadingOverlay{},
		&KeyboardHandler{},
	)
}

// WithMinimalLayout wraps page content without Navbar (e.g. for login page).
func WithMinimalLayout(content ...app.UI) app.UI {
	return app.Div().Class("min-h-screen bg-gradient-to-br from-base-100 via-base-200/20 to-base-100 flex flex-col").Body(
		app.Main().Class("flex-1 container mx-auto px-4 sm:px-6 py-8 max-w-7xl").Body(
			content...,
		),

		&Toast{},
		&LoadingOverlay{},
	)
}
