package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	FlexClass       = "uk-flex"
	FlexInlineClass = "uk-flex-inline"

	FlexLeftClass    = "uk-flex-left"
	FlexCenterClass  = "uk-flex-center"
	FlexRightClass   = "uk-flex-right"
	FlexBetweenClass = "uk-flex-between"
	FlexAroundClass  = "uk-flex-around"

	FlexRowClass           = "uk-flex-row"
	FlexRowReverseClass    = "uk-flex-row-reverse"
	FlexColumnClass        = "uk-flex-column"
	FlexColumnReverseClass = "uk-flex-column-reverse"

	FlexWrapClass        = "uk-flex-wrap"
	FlexWrapReverseClass = "uk-flex-wrap-reverse"
	FlexNowrapClass      = "uk-flex-nowrap"
)

func Flex(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class("uk-flex").Body(elems...)
}
