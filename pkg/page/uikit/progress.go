package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

func Progress(value, max int) app.HTMLProgress {
	return app.Progress().Class("uk-progress").Value(value).Max(max)
}
