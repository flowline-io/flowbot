package component

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/page/form"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"net/http"
)

type Form struct {
	app.Compo
	Page   model.Page
	Form   model.Form
	Schema types.FormMsg
}

func (c *Form) Render() app.UI {
	var fields []app.UI

	var alert app.UI
	switch c.Page.State {
	case model.PageStateProcessedSuccess:
		alert = app.Div().Class("uk-alert-success").Body(
			app.P().Style("padding", "20px").Text(fmt.Sprintf("Form [%s] processed success, %s", c.Page.PageID, c.Page.UpdatedAt)))
	case model.PageStateProcessedFailed:
		alert = app.Div().Class("uk-alert-danger").Body(
			app.P().Style("padding", "20px").Text(fmt.Sprintf("Form [%s] processed failed, %s", c.Page.PageID, c.Page.UpdatedAt)))
	}

	// hidden
	fields = append(fields, app.Input().Hidden(true).Type("text").Name("x-csrf-token").Value(types.Id()))
	fields = append(fields, app.Input().Hidden(true).Type("text").Name("x-form_id").Value(c.Page.PageID))
	fields = append(fields, app.Input().Hidden(true).Type("text").Name("x-uid").Value(c.Page.UID))
	fields = append(fields, app.Input().Hidden(true).Type("text").Name("x-topic").Value(c.Page.Topic))

	// fields
	builder := form.NewBuilder(c.Schema.Field)
	builder.Method = http.MethodPost
	builder.Action = "/form"
	// button
	if c.Page.State == model.PageStateCreated {
		builder.Button = []app.UI{
			app.Div().Class("uk-margin").Body(
				app.Button().Class("uk-button uk-button-primary").Text("Submit").Type("submit"),
			),
		}
	}
	ui, err := builder.UI()
	if err != nil {
		return nil
	}
	fields = append(fields, ui)

	// record value
	if c.Page.State == model.PageStateProcessedSuccess || c.Page.State == model.PageStateProcessedFailed {
		fields = append(fields, app.Div().Class("").Body(
			app.H3().Text("Submit values"),
			app.Pre().Text(c.Form.Values),
		))
	}

	return app.Div().Body(
		alert,
		app.H1().Class(".uk-heading-small").Text(c.Schema.Title),
		app.Form().Class("uk-form-stacked").Method("POST").Action("/form").
			Body(fields...),
	)
}
