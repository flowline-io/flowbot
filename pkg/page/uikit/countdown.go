package uikit

import (
	"fmt"
	"time"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
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
