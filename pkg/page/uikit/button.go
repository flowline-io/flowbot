package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	ButtonDefaultClass   = "uk-button-default"
	ButtonPrimaryClass   = "uk-button-primary"
	ButtonSecondaryClass = "uk-button-secondary"
	ButtonDangerClass    = "uk-button-danger"
	ButtonTextClass      = "uk-button-text"
	ButtonLinkClass      = "uk-button-link"

	ButtonSmallClass = "uk-button-small"
	ButtonLargeClass = "uk-button-large"
)

func Button(text string) app.HTMLButton {
	return app.Button().Class("uk-button").Text(text)
}

func ButtonGroup(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class("uk-button-group").Body(elems...)
}
