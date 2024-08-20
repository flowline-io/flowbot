package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

func Icon(name string) app.HTMLSpan {
	return app.Span().Attr("uk-icon", name)
}
