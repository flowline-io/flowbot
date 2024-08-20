package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

func Grid(elems ...app.UI) app.HTMLDiv {
	return app.Div().Attr("uk-grid", "").Body(elems...)
}
