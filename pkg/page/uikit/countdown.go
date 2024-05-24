package uikit

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"time"
)

func Countdown(datetime time.Time) app.HTMLDiv {
	return Grid(
		Div(
			Div().Class("uk-countdown-number uk-countdown-days"),
		),
		Div().Class("uk-countdown-separator").Text(":"),
		Div(
			Div().Class("uk-countdown-number uk-countdown-hours"),
		),
		Div().Class("uk-countdown-separator").Text(":"),
		Div(
			Div().Class("uk-countdown-number uk-countdown-minutes"),
		),
		Div().Class("uk-countdown-separator").Text(":"),
		Div(
			Div().Class("uk-countdown-number uk-countdown-seconds"),
		),
	).Class("uk-grid-small uk-child-width-auto uk-margin").
		Attr("uk-countdown", fmt.Sprintf("date: %s", datetime.Format(time.RFC3339)))
}
