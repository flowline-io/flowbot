package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

func Link(title, url string) app.HTMLA {
	return app.A().Class("uk-link-muted").Href(url).Text(title)
}
