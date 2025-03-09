package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

const (
	NavbarClass            = "uk-navbar"
	NavbarContainerClass   = "uk-navbar-container"
	NavbarLeftClass        = "uk-navbar-left"
	NavbarRightClass       = "uk-navbar-right"
	NavbarCenterClass      = "uk-navbar-center"
	NavbarItemClass        = "uk-navbar-item"
	NavbarNavClass         = "uk-navbar-nav"
	NavbarDropdownClass    = "uk-navbar-dropdown"
	NavbarTransparentClass = "uk-navbar-transparent"
	NavbarStickyClass      = "uk-navbar-sticky"
	NavbarSubtitleClass    = "uk-navbar-subtitle"
	NavbarToggleClass      = "uk-navbar-toggle"
	NavbarDropNavClass     = "uk-navbar-dropdown-nav"
)

// Navbar creates a basic navigation bar
func Navbar(elems ...app.UI) app.HTMLNav {
	return app.Nav().Class(NavbarContainerClass).Body(
		app.Div().Class(NavbarClass).Body(elems...),
	)
}

// NavbarLeft creates the left part of the navigation bar
func NavbarLeft(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class(NavbarLeftClass).Body(elems...)
}

// NavbarRight creates the right part of the navigation bar
func NavbarRight(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class(NavbarRightClass).Body(elems...)
}

// NavbarCenter creates the center part of the navigation bar
func NavbarCenter(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class(NavbarCenterClass).Body(elems...)
}

// NavbarNav creates a container for navigation items in the navigation bar
func NavbarNav(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(NavbarNavClass).Body(elems...)
}

// NavbarItem creates a navigation item in the navigation bar
func NavbarItem(text string, href string) app.HTMLLi {
	return app.Li().Body(
		app.A().Href(href).Text(text),
	)
}

// NavbarLogo creates a logo item in the navigation bar
func NavbarLogo(text string, href string) app.HTMLDiv {
	return app.Div().Class(NavbarItemClass).Body(
		app.A().Class("uk-logo").Href(href).Text(text),
	)
}

// NavbarToggle creates a toggle button for the navigation bar (for mobile)
func NavbarToggle(target string) app.HTMLA {
	return app.A().Class(NavbarToggleClass).Attr("uk-navbar-toggle-icon", "").Href("#").Attr("uk-toggle", "target: "+target)
}
