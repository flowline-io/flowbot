package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

func Badge(number int) app.HTMLSpan {
	return app.Span().Class("uk-badge").Text(number)
}
