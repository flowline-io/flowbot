package component

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Html struct {
	app.Compo
	Page   gen.Page
	Schema types.HtmlMsg
}

func (c *Html) Render() app.UI {
	return app.Raw(c.Schema.Raw)
}
