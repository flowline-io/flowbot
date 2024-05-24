package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	MarginClass       = "uk-margin"
	MarginTopClass    = "uk-margin-top"
	MarginBottomClass = "uk-margin-bottom"
	MarginLeftClass   = "uk-margin-left"
	MarginRightClass  = "uk-margin-right"
)

func Margin(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class("uk-margin").Body(elems...)
}
