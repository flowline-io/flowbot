package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

func Container(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class("uk-container").Body(elems...)
}
