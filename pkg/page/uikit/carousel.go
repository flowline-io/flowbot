package uikit

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const (
	SliderClass          = "uk-slider"
	SliderContainerClass = "uk-slider-container"
	SliderItemsClass     = "uk-slider-items"
	SliderNavClass       = "uk-slider-nav"
	SliderDotsClass      = "uk-dotnav"

	SlideshowClass      = "uk-slideshow"
	SlideshowItemsClass = "uk-slideshow-items"
	SlideshowNavClass   = "uk-slideshow-nav"

	CarouselClass = "uk-carousel"
)

// Slider creates a basic slider component
func Slider(id string, elems ...app.UI) app.HTMLDiv {
	return app.Div().ID(id).Class(SliderClass).Attr("uk-slider", "").Body(
		app.Div().Class(SliderContainerClass).Body(
			app.Ul().Class(SliderItemsClass, "uk-child-width-1-1").Body(elems...),
		),
	)
}

// SliderWithOptions creates a slider component with options
func SliderWithOptions(id string, autoplay bool, autoplayInterval int, center bool, finite bool, elems ...app.UI) app.HTMLDiv {
	options := ""

	if autoplay {
		options += "autoplay: true; "
	}

	if autoplayInterval > 0 {
		options += fmt.Sprintf("autoplay-interval: %d; ", autoplayInterval)
	}

	if center {
		options += "center: true; "
	}

	if finite {
		options += "finite: true; "
	}

	return app.Div().ID(id).Class(SliderClass).Attr("uk-slider", options).Body(
		app.Div().Class(SliderContainerClass).Body(
			app.Ul().Class(SliderItemsClass, "uk-child-width-1-1").Body(elems...),
		),
	)
}

// SliderItem creates a slider item
func SliderItem(content app.UI) app.HTMLLi {
	return app.Li().Body(content)
}

// SliderWithNav creates a slider component with navigation
func SliderWithNav(id string, items []app.UI) app.HTMLDiv {
	return app.Div().Body(
		Slider(id, items...),
		app.Ul().Class(SliderNavClass, SliderDotsClass, "uk-margin").Attr("uk-slider-nav", ""),
	)
}

// Slideshow creates a basic slideshow component
func Slideshow(id string, elems ...app.UI) app.HTMLDiv {
	return app.Div().ID(id).Class(SlideshowClass).Attr("uk-slideshow", "").Body(
		app.Ul().Class(SlideshowItemsClass).Body(elems...),
	)
}

// SlideshowWithOptions creates a slideshow component with options
func SlideshowWithOptions(id string, autoplay bool, autoplayInterval int, pauseOnHover bool, ratio string, elems ...app.UI) app.HTMLDiv {
	options := ""

	if autoplay {
		options += "autoplay: true; "
	}

	if autoplayInterval > 0 {
		options += fmt.Sprintf("autoplay-interval: %d; ", autoplayInterval)
	}

	if !pauseOnHover {
		options += "pause-on-hover: false; "
	}

	if ratio != "" {
		options += fmt.Sprintf("ratio: %s; ", ratio)
	}

	return app.Div().ID(id).Class(SlideshowClass).Attr("uk-slideshow", options).Body(
		app.Ul().Class(SlideshowItemsClass).Body(elems...),
	)
}

// SlideshowItem creates a slideshow item
func SlideshowItem(content app.UI) app.HTMLLi {
	return app.Li().Body(content)
}

// SlideshowWithNav creates a slideshow component with navigation
func SlideshowWithNav(id string, items []app.UI) app.HTMLDiv {
	return app.Div().Body(
		Slideshow(id, items...),
		app.Ul().Class(SlideshowNavClass, SliderDotsClass, "uk-margin").Attr("uk-slideshow-nav", ""),
	)
}

// SlideshowImage creates a slideshow image item
func SlideshowImage(src string, alt string) app.HTMLLi {
	return app.Li().Body(
		app.Img().Src(src).Alt(alt).Attr("uk-cover", ""),
	)
}
