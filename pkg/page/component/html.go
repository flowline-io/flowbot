package component

import (
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Html struct {
	app.Compo
	Page   model.Page
	Schema types.HtmlMsg
}

func (c *Html) Render() app.UI {
	return app.Raw(c.Schema.Raw)
}
