package components

import (
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Breadcrumb struct {
	app.Compo

	items []BreadcrumbItem
}

type BreadcrumbItem struct {
	Label string
	Href  string
}

func (b *Breadcrumb) OnNav(ctx app.Context) {
	b.items = buildBreadcrumbFromPath(ctx.Page().URL().Path)
}

func buildBreadcrumbFromPath(path string) []BreadcrumbItem {
	path = strings.TrimPrefix(path, "/admin")
	if path == "" || path == "/" {
		return []BreadcrumbItem{{Label: "Home", Href: "/admin"}}
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	items := []BreadcrumbItem{{Label: "Home", Href: "/admin"}}

	accumulatedPath := "/admin"
	for i, segment := range segments {
		if segment == "" {
			continue
		}
		accumulatedPath += "/" + segment
		label := formatBreadcrumbLabel(segment)
		if i == len(segments)-1 {
			items = append(items, BreadcrumbItem{Label: label, Href: ""})
		} else {
			items = append(items, BreadcrumbItem{Label: label, Href: accumulatedPath})
		}
	}

	return items
}

func formatBreadcrumbLabel(segment string) string {
	switch segment {
	case "users":
		return "Users"
	case "containers":
		return "Containers"
	case "workflows":
		return "Workflows"
	case "bots":
		return "Bots"
	case "logs":
		return "Logs"
	case "settings":
		return "Settings"
	case "login":
		return "Login"
	default:
		return segment
	}
}

func (b *Breadcrumb) Render() app.UI {
	if len(b.items) <= 1 {
		return app.Div()
	}

	items := make([]app.UI, 0, len(b.items)*2)
	for i, item := range b.items {
		if i > 0 {
			items = append(items, app.Li().Class("text-base-content/40").Text("/"))
		}

		if item.Href == "" || i == len(b.items)-1 {
			items = append(items, app.Li().Class("text-base-content/70 font-medium").Text(item.Label))
		} else {
			items = append(items, app.Li().Body(
				app.A().Href(item.Href).Class("link link-hover text-base-content/60 hover:text-base-content").Text(item.Label),
			))
		}
	}

	return app.Div().Class("text-sm breadcrumbs py-2 px-1").Body(
		app.Ul().Class("flex items-center gap-1").Body(items...),
	)
}
