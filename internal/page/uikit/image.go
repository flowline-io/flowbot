package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

func Image(src string) app.HTMLDiv {
	return app.Div().
		Attr("uk-img", "loading: eager").
		Class("uk-height-medium uk-flex uk-flex-center uk-flex-middle uk-background-cover uk-light").
		DataSet("src", src)
}
