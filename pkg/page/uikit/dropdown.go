package uikit

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"strconv"
)

const (
	DropdownClass          = "uk-dropdown"
	DropdownNavClass       = "uk-dropdown-nav"
	DropdownGridClass      = "uk-dropdown-grid"
	DropdownCloseClass     = "uk-dropdown-close"
	DropdownScrollClass    = "uk-dropdown-scroll"
	DropdownSmallClass     = "uk-dropdown-small"
	DropdownLargeClass     = "uk-dropdown-large"
	DropdownWidthSmall     = "small"
	DropdownWidthMedium    = "medium"
	DropdownWidthLarge     = "large"
	DropdownWidthXLarge    = "xlarge"
	DropdownPosBottom      = "bottom-left"
	DropdownPosBottomRight = "bottom-right"
	DropdownPosTop         = "top-left"
	DropdownPosTopRight    = "top-right"
	DropdownPosLeft        = "left-top"
	DropdownPosLeftBottom  = "left-bottom"
	DropdownPosRight       = "right-top"
	DropdownPosRightBottom = "right-bottom"
)

// Dropdown creates a basic dropdown menu
func Dropdown(id string, elems ...app.UI) app.HTMLDiv {
	return app.Div().ID(id).Class(DropdownClass).Attr("uk-dropdown", "").Body(elems...)
}

// DropdownWithOptions creates a dropdown menu with options
func DropdownWithOptions(id string, mode string, pos string, offset int, animation bool, duration int, elems ...app.UI) app.HTMLDiv {
	options := ""
	if mode != "" {
		options += "mode: " + mode + "; "
	}
	if pos != "" {
		options += "pos: " + pos + "; "
	}
	if offset != 0 {
		options += "offset: " + strconv.Itoa(offset) + "; "
	}
	if !animation {
		options += "animation: false; "
	}
	if duration != 0 {
		options += "duration: " + strconv.Itoa(duration) + "; "
	}

	return app.Div().ID(id).Class(DropdownClass).Attr("uk-dropdown", options).Body(elems...)
}

// DropdownNav creates dropdown menu navigation
func DropdownNav(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(DropdownNavClass).Body(elems...)
}

// DropdownItem creates a dropdown menu item
func DropdownItem(text string, href string) app.HTMLLi {
	return app.Li().Body(
		app.A().Href(href).Text(text),
	)
}

// DropdownDivider creates a dropdown menu divider
func DropdownDivider() app.HTMLLi {
	return app.Li().Class("uk-nav-divider")
}

// DropdownHeader creates a dropdown menu header
func DropdownHeader(text string) app.HTMLLi {
	return app.Li().Class("uk-nav-header").Text(text)
}

// DropdownButton creates a button with a dropdown menu
func DropdownButton(text string, dropdownID string) app.HTMLButton {
	return Button(text).Attr("uk-toggle", "target: #"+dropdownID)
}
