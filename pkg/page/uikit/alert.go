package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	AlertCloseClass   = "uk-alert-close"
	AlertPrimaryClass = "uk-alert-primary"
	AlertSuccessClass = "uk-alert-success"
	AlertWarningClass = "uk-alert-warning"
	AlertDangerClass  = "uk-alert-danger"
)

func Alert(elems ...app.UI) app.HTMLDiv {
	return app.Div().Attr("uk-alert", "").Body(elems...)
}
