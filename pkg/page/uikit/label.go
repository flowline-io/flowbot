package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	LabelSuccessClass = "uk-label-success"
	LabelWarningClass = "uk-label-warning"
	LabelDangerClass  = "uk-label-danger"
)

func Label(text string) app.HTMLSpan {
	return app.Span().Class("uk-label").Text(text)
}
