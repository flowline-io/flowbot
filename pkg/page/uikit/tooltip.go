package uikit

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const (
	TooltipPosTop         = "top"
	TooltipPosTopLeft     = "top-left"
	TooltipPosTopRight    = "top-right"
	TooltipPosBottom      = "bottom"
	TooltipPosBottomLeft  = "bottom-left"
	TooltipPosBottomRight = "bottom-right"
	TooltipPosLeft        = "left"
	TooltipPosRight       = "right"
)

// WithTooltip adds a tooltip to any UI element
func WithTooltip(element app.UI, title string) app.UI {
	switch e := element.(type) {
	case app.HTMLDiv:
		return e.Attr("uk-tooltip", title)
	case app.HTMLButton:
		return e.Attr("uk-tooltip", title)
	case app.HTMLA:
		return e.Attr("uk-tooltip", title)
	case app.HTMLSpan:
		return e.Attr("uk-tooltip", title)
	default:
		return element
	}
}

// WithTooltipPos adds a tooltip with position to any UI element
func WithTooltipPos(element app.UI, title string, pos string) app.UI {
	tooltip := fmt.Sprintf("title: %s; pos: %s", title, pos)

	switch e := element.(type) {
	case app.HTMLDiv:
		return e.Attr("uk-tooltip", tooltip)
	case app.HTMLButton:
		return e.Attr("uk-tooltip", tooltip)
	case app.HTMLA:
		return e.Attr("uk-tooltip", tooltip)
	case app.HTMLSpan:
		return e.Attr("uk-tooltip", tooltip)
	default:
		return element
	}
}

// WithTooltipOptions adds a tooltip with complete options to any UI element
func WithTooltipOptions(element app.UI, title string, pos string, offset int, animation bool, duration int, delay int) app.UI {
	options := fmt.Sprintf("title: %s", title)

	if pos != "" {
		options += fmt.Sprintf("; pos: %s", pos)
	}

	if offset != 0 {
		options += fmt.Sprintf("; offset: %d", offset)
	}

	if !animation {
		options += "; animation: false"
	}

	if duration != 0 {
		options += fmt.Sprintf("; duration: %d", duration)
	}

	if delay != 0 {
		options += fmt.Sprintf("; delay: %d", delay)
	}

	switch e := element.(type) {
	case app.HTMLDiv:
		return e.Attr("uk-tooltip", options)
	case app.HTMLButton:
		return e.Attr("uk-tooltip", options)
	case app.HTMLA:
		return e.Attr("uk-tooltip", options)
	case app.HTMLSpan:
		return e.Attr("uk-tooltip", options)
	default:
		return element
	}
}
