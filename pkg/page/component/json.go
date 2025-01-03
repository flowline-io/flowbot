package component

import (
	"fmt"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Json struct {
	app.Compo
	Page   model.Page
	Schema types.TextMsg
}

func (c *Json) Render() app.UI {
	return app.Raw(fmt.Sprintf(`
<div id="json-viewer"></div>
<script src="https://cdn.jsdelivr.net/npm/@textea/json-viewer"></script>
<script>
  new JsonViewer({
    value: %s
  }).render()
</script>
`, c.Schema.Text))
}
