package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

const (
	TabsClass           = "uk-tab"
	TabsLeftClass       = "uk-tab-left"
	TabsRightClass      = "uk-tab-right"
	TabsBottomClass     = "uk-tab-bottom"
	TabsJustifiedClass  = "uk-tab-justified"
	SwitcherClass       = "uk-switcher"
	SwitcherItemClass   = "uk-switcher-item"
	TabsResponsiveClass = "uk-tab-responsive"
	ActiveClass         = "uk-active"
)

// Tabs creates a tab navigation
func Tabs(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(TabsClass).Attr("uk-tab", "").Body(elems...)
}

// TabsLeft creates a left tab navigation
func TabsLeft(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(TabsClass, TabsLeftClass).Attr("uk-tab", "").Body(elems...)
}

// TabsRight creates a right tab navigation
func TabsRight(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(TabsClass, TabsRightClass).Attr("uk-tab", "").Body(elems...)
}

// TabsBottom creates a bottom tab navigation
func TabsBottom(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(TabsClass, TabsBottomClass).Attr("uk-tab", "").Body(elems...)
}

// TabItem creates a tab item
func TabItem(text string, active bool) app.HTMLLi {
	li := app.Li()
	if active {
		li = li.Class(ActiveClass)
	}
	return li.Body(
		app.A().Href("#").Text(text),
	)
}

// Switcher creates a tab content switcher
func Switcher(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(SwitcherClass).Attr("uk-switcher", "animation: uk-animation-fade").Body(elems...)
}

// SwitcherItem creates a tab content item
func SwitcherItem(active bool, elems ...app.UI) app.HTMLLi {
	li := app.Li()
	if active {
		li = li.Class(ActiveClass)
	}
	return li.Body(elems...)
}

// TabsWithContent creates tabs with content (composite usage)
func TabsWithContent(tabsID string, contentID string, items []struct {
	Title   string
	Content app.UI
	Active  bool
}) app.HTMLDiv {
	var tabItems []app.UI
	var contentItems []app.UI

	for _, item := range items {
		tabItems = append(tabItems, TabItem(item.Title, item.Active))
		contentItems = append(contentItems, SwitcherItem(item.Active, item.Content))
	}

	return app.Div().Body(
		app.Ul().ID(tabsID).Class(TabsClass).Attr("uk-tab", "connect: #"+contentID).Body(tabItems...),
		app.Ul().ID(contentID).Class(SwitcherClass).Body(contentItems...),
	)
}
