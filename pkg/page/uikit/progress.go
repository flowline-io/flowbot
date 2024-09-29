package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

func Progress(value, maxValue int) app.HTMLProgress {
	return app.Progress().Class("uk-progress").Value(value).Max(maxValue)
}
