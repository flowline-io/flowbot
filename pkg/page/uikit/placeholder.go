package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

func Placeholder(text string) app.HTMLDiv {
	return app.Div().Class("uk-placeholder uk-text-center").Text(text)
}
