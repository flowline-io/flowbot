package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

func Card(title string, body ...app.UI) app.HTMLDiv {
	var elems []app.UI
	if title != "" {
		elems = append(elems, app.Div().Class("uk-card-title").Text(title))
	}
	elems = append(elems, body...)
	return app.Div().Class("uk-card uk-card-body").
		Body(elems...)
}
