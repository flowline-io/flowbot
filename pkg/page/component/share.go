package component

import (
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type Share struct {
	app.Compo
	Page   model.Page
	Schema types.TextMsg
}

func (c *Share) Render() app.UI {
	return app.Div().Body(
		app.H1().Class(".uk-heading-small").Text("Share"),
		app.Code().Text(c.Schema.Text),
	)
}
