package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	ListDiscClass    = "uk-list-disc"
	ListCircleClass  = "uk-list-circle"
	ListSquareClass  = "uk-list-square"
	ListDecimalClass = "uk-list-decimal"
	ListHyphenClass  = "uk-list-hyphen"
)

func List(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class("uk-list").Body(elems...)
}
