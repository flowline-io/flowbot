package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

func DividerIcon() app.HTMLHr {
	return app.Hr().Class("uk-divider-icon")
}

func DividerSmall() app.HTMLHr {
	return app.Hr().Class("uk-divider-small")
}

func DividerVertical() app.HTMLHr {
	return app.Hr().Class("uk-divider-vertical")
}
