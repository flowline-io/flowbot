package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	PaddingClass      = "uk-padding"
	PaddingSmallClass = "uk-padding-small"
	PaddingLargeClass = "uk-padding-large"
)

func Padding(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class("uk-padding")
}
