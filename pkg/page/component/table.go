package component

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"math"
)

type Table struct {
	app.Compo
	Page   model.Page
	Schema types.TableMsg
}

func (c *Table) Render() app.UI {
	var alert app.UI
	switch c.Page.State {
	case model.PageStateProcessedSuccess:
		alert = app.Div().Class("uk-alert-success").Body(
			app.P().Style("padding", "20px").Text(fmt.Sprintf("Table [%s] processed success, %s", c.Page.PageID, c.Page.UpdatedAt)))
	case model.PageStateProcessedFailed:
		alert = app.Div().Class("uk-alert-danger").Body(
			app.P().Style("padding", "20px").Text(fmt.Sprintf("Table [%s] processed failed, %s", c.Page.PageID, c.Page.UpdatedAt)))
	}

	return app.Div().Body(
		alert,
		app.H1().Class(".uk-heading-small").Text(c.Schema.Title),
		app.Table().Class("uk-table uk-table-striped").Body(
			app.THead().Body(
				app.Tr().Body(
					app.Range(c.Schema.Header).Slice(func(i int) app.UI {
						return app.Th().Text(c.Schema.Header[i])
					}),
				),
			),
			app.TBody().Body(
				app.Range(c.Schema.Row).Slice(func(i int) app.UI {
					return app.Tr().Body(
						app.Range(c.Schema.Row[i]).Slice(func(j int) app.UI {
							item := c.Schema.Row[i][j]
							if txt, ok := item.(string); ok && utils.IsUrl(txt) {
								return app.Td().Body(app.A().Target("_blank").Href(txt).Text(txt))
							} else if num, ok := item.(float64); ok {
								_, frac := math.Modf(num)
								if frac == 0 {
									return app.Td().Text(int(num))
								} else {
									return app.Td().Text(num)
								}
							} else {
								return app.Td().Text(c.Schema.Row[i][j])
							}
						}),
					)
				}),
			),
		),
	)
}
