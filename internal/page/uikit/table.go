package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	TableDividerClass    = "uk-table-divider"
	TableStripedClass    = "uk-table-striped"
	TableHoverClass      = "uk-table-hover"
	TableSmallClass      = "uk-table-small"
	TableLargeClass      = "uk-table-large"
	TableJustifyClass    = "uk-table-justify"
	TableMiddleClass     = "uk-table-middle"
	TableResponsiveClass = "uk-table-responsive"
)

func Table(elems ...app.UI) app.HTMLTable {
	return app.Table().Class("uk-table").Body(elems...)
}

func THead(elems ...app.UI) app.HTMLTHead {
	return app.THead().Body(elems...)
}

func TBody(elems ...app.UI) app.HTMLTBody {
	return app.TBody().Body(elems...)
}

func TFoot(elems ...app.UI) app.HTMLTFoot {
	return app.TFoot().Body(elems...)
}

func Tr(elems ...app.UI) app.HTMLTr {
	return app.Tr().Body(elems...)
}

func Th(elems ...app.UI) app.HTMLTh {
	return app.Th().Body(elems...)
}

func Td(elems ...app.UI) app.HTMLTd {
	return app.Td().Body(elems...)
}
