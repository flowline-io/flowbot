package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

func Article(title string, meta string, body ...app.UI) app.HTMLArticle {
	var elems []app.UI
	if title != "" {
		elems = append(elems, app.H1().Text(title))
	}
	if meta != "" {
		elems = append(elems, app.P().Text(meta))
	}
	elems = append(elems, body...)
	return app.Article().Class("uk-article").Body(elems...)
}
