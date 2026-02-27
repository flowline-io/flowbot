// Package components provides reusable UI components for the Admin frontend.
package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// WithLayout wraps page content in a unified layout skeleton: Navbar + main content area + Toast.
// Every page's Render() method should call this function to maintain a consistent visual structure.
func WithLayout(content ...app.UI) app.UI {
	return app.Div().Class("min-h-screen bg-base-200 flex flex-col").Body(
		// Top navigation bar
		&Navbar{},

		// Main content area
		app.Main().Class("flex-1 container mx-auto px-4 py-6 max-w-7xl").Body(
			content...,
		),

		// Global toast notification container
		&Toast{},
	)
}
