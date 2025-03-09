package uikit

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"strconv"
)

const (
	AccordionClass        = "uk-accordion"
	AccordionTitleClass   = "uk-accordion-title"
	AccordionContentClass = "uk-accordion-content"
	AccordionOpenClass    = "uk-open"
)

// Accordion 创建手风琴组件
func Accordion(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(AccordionClass).Attr("uk-accordion", "").Body(elems...)
}

// AccordionWithOptions 创建带选项的手风琴组件
func AccordionWithOptions(multiple bool, collapsible bool, animation bool, duration int, elems ...app.UI) app.HTMLUl {
	options := ""

	if multiple {
		options += "multiple: true; "
	}

	if collapsible {
		options += "collapsible: true; "
	}

	if !animation {
		options += "animation: false; "
	}

	if duration != 0 {
		options += "duration: " + strconv.Itoa(duration) + "; "
	}

	return app.Ul().Class(AccordionClass).Attr("uk-accordion", options).Body(elems...)
}

// AccordionItem 创建手风琴项
func AccordionItem(title string, content app.UI, open bool) app.HTMLLi {
	li := app.Li()
	if open {
		li = li.Class(AccordionOpenClass)
	}

	return li.Body(
		app.A().Class(AccordionTitleClass).Href("#").Text(title),
		app.Div().Class(AccordionContentClass).Body(content),
	)
}

// AccordionItems 创建多个手风琴项
func AccordionItems(items []struct {
	Title   string
	Content app.UI
	Open    bool
}) []app.UI {
	var result []app.UI

	for _, item := range items {
		result = append(result, AccordionItem(item.Title, item.Content, item.Open))
	}

	return result
}
