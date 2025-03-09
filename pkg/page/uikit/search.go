package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

const (
	SearchClass        = "uk-search"
	SearchDefaultClass = "uk-search-default"
	SearchLargeClass   = "uk-search-large"
	SearchNavbarClass  = "uk-search-navbar"
	SearchIconClass    = "uk-search-icon"
	SearchInputClass   = "uk-search-input"
	SearchToggleClass  = "uk-search-toggle"
)

// Search creates a basic search box
func Search(id string, name string, placeholder string) app.HTMLForm {
	return app.Form().Class(SearchClass, SearchDefaultClass).Body(
		app.Span().Class(SearchIconClass).Attr("uk-search-icon", ""),
		app.Input().
			ID(id).
			Name(name).
			Class(SearchInputClass).
			Type("search").
			Placeholder(placeholder),
	)
}

// SearchLarge creates a large-sized search box
func SearchLarge(id string, name string, placeholder string) app.HTMLForm {
	return app.Form().Class(SearchClass, SearchLargeClass).Body(
		app.Span().Class(SearchIconClass).Attr("uk-search-icon", ""),
		app.Input().
			ID(id).
			Name(name).
			Class(SearchInputClass).
			Type("search").
			Placeholder(placeholder),
	)
}

// SearchNavbar creates a navigation bar search box
func SearchNavbar(id string, name string, placeholder string) app.HTMLForm {
	return app.Form().Class(SearchClass, SearchNavbarClass).Body(
		app.Span().Class(SearchIconClass).Attr("uk-search-icon", ""),
		app.Input().
			ID(id).
			Name(name).
			Class(SearchInputClass).
			Type("search").
			Placeholder(placeholder),
	)
}

// SearchWithButton creates a search box with a button
func SearchWithButton(id string, name string, placeholder string, buttonText string) app.HTMLForm {
	return app.Form().Class("uk-search uk-search-default uk-width-medium").Body(
		app.Input().
			ID(id).
			Name(name).
			Class(SearchInputClass).
			Type("search").
			Placeholder(placeholder),
		app.Button().Type("submit").Class("uk-search-icon uk-search-icon-flip").Attr("uk-search-icon", ""),
	)
}

// SearchWithDropdown creates a search box with a dropdown menu
func SearchWithDropdown(id string, name string, placeholder string, dropdownID string, dropdownItems []app.UI) app.HTMLDiv {
	return app.Div().Body(
		app.Form().Class(SearchClass, SearchDefaultClass).Body(
			app.Span().Class(SearchIconClass).Attr("uk-search-icon", ""),
			app.Input().
				ID(id).
				Name(name).
				Class(SearchInputClass).
				Type("search").
				Placeholder(placeholder).
				Attr("uk-toggle", "target: #"+dropdownID),
		),
		app.Div().ID(dropdownID).Class("uk-dropdown").Attr("uk-dropdown", "mode: click; pos: bottom-left").Body(
			app.Ul().Class("uk-nav uk-dropdown-nav").Body(dropdownItems...),
		),
	)
}

// SearchToggle creates a search toggle button
func SearchToggle(targetID string) app.HTMLA {
	return app.A().Class(SearchToggleClass).Href("#").Attr("uk-search-icon", "").Attr("uk-toggle", "target: #"+targetID)
}

// SearchModal creates a modal search box
func SearchModal(id string, name string, placeholder string) app.HTMLDiv {
	return app.Div().ID(id).Class("uk-modal-full uk-modal").Attr("uk-modal", "").Body(
		app.Div().Class("uk-modal-dialog uk-flex uk-flex-center uk-flex-middle").Attr("uk-height-viewport", "").Body(
			app.Button().Class("uk-modal-close-full").Type("button").Attr("uk-close", ""),
			app.Form().Class("uk-search uk-search-large").Body(
				app.Input().
					Class("uk-search-input uk-text-center").
					Name(name).
					Type("search").
					Placeholder(placeholder).
					Attr("autofocus", ""),
			),
		),
	)
}
