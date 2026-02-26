package uikit

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const (
	PaginationClass       = "uk-pagination"
	PaginationPrevClass   = "uk-pagination-previous"
	PaginationNextClass   = "uk-pagination-next"
	PaginationLeftClass   = "uk-flex-left"
	PaginationCenterClass = "uk-flex-center"
	PaginationRightClass  = "uk-flex-right"
	PaginationLargeClass  = "uk-pagination-large"
)

// Pagination creates a basic pagination component
func Pagination(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(PaginationClass).Body(elems...)
}

// PaginationCenter creates a centered pagination component
func PaginationCenter(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(PaginationClass, PaginationCenterClass).Body(elems...)
}

// PaginationRight creates a right-aligned pagination component
func PaginationRight(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(PaginationClass, PaginationRightClass).Body(elems...)
}

// PaginationLarge creates a large-sized pagination component
func PaginationLarge(elems ...app.UI) app.HTMLUl {
	return app.Ul().Class(PaginationClass, PaginationLargeClass).Body(elems...)
}

// PaginationItem creates a pagination item
func PaginationItem(page int, href string, active bool) app.HTMLLi {
	li := app.Li()
	if active {
		li = li.Class(ActiveClass)
	}

	return li.Body(
		app.A().Href(href).Text(fmt.Sprintf("%d", page)),
	)
}

// PaginationPrevious creates a previous page button
func PaginationPrevious(href string, disabled bool) app.HTMLLi {
	li := app.Li().Class(PaginationPrevClass)

	a := app.A().Href(href)
	if disabled {
		a = a.Attr("aria-disabled", "true").Class(DisabledClass)
	}

	return li.Body(
		a.Body(
			app.Span().Attr("uk-pagination-previous", ""),
			app.Span().Text("Previous"),
		),
	)
}

// PaginationNext creates a next page button
func PaginationNext(href string, disabled bool) app.HTMLLi {
	li := app.Li().Class(PaginationNextClass)

	a := app.A().Href(href)
	if disabled {
		a = a.Attr("aria-disabled", "true").Class(DisabledClass)
	}

	return li.Body(
		a.Body(
			app.Span().Text("Next"),
			app.Span().Attr("uk-pagination-next", ""),
		),
	)
}

// PaginationEllipsis creates an ellipsis
func PaginationEllipsis() app.HTMLLi {
	return app.Li().Class("uk-disabled").Body(
		app.Span().Text("..."),
	)
}

// CreatePagination creates a complete pagination component
func CreatePagination(currentPage, totalPages int, urlFormat string, maxVisible int) app.HTMLUl {
	var items []app.UI

	// Add previous page button
	prevDisabled := currentPage <= 1
	prevHref := "#"
	if !prevDisabled {
		prevHref = fmt.Sprintf(urlFormat, currentPage-1)
	}
	items = append(items, PaginationPrevious(prevHref, prevDisabled))

	// Calculate the range of displayed page numbers
	startPage := 1
	endPage := totalPages

	if maxVisible > 0 && totalPages > maxVisible {
		// Calculate start and end page numbers
		half := maxVisible / 2
		startPage = max(currentPage-half, 1)

		endPage = startPage + maxVisible - 1
		if endPage > totalPages {
			endPage = totalPages
			startPage = max(endPage-maxVisible+1, 1)
		}
	}

	// Add first page (if not in range)
	if startPage > 1 {
		items = append(items, PaginationItem(1, fmt.Sprintf(urlFormat, 1), false))

		// Add ellipsis (if needed)
		if startPage > 2 {
			items = append(items, PaginationEllipsis())
		}
	}

	// Add page numbers
	for i := startPage; i <= endPage; i++ {
		items = append(items, PaginationItem(i, fmt.Sprintf(urlFormat, i), i == currentPage))
	}

	// Add last page (if not in range)
	if endPage < totalPages {
		// Add ellipsis (if needed)
		if endPage < totalPages-1 {
			items = append(items, PaginationEllipsis())
		}

		items = append(items, PaginationItem(totalPages, fmt.Sprintf(urlFormat, totalPages), false))
	}

	// Add next page button
	nextDisabled := currentPage >= totalPages
	nextHref := "#"
	if !nextDisabled {
		nextHref = fmt.Sprintf(urlFormat, currentPage+1)
	}
	items = append(items, PaginationNext(nextHref, nextDisabled))

	return PaginationCenter(items...)
}
