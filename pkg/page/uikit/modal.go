package uikit

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

const (
	ModalCloseClass = "uk-modal-close"
)

func ModalToggle(id, title string) app.HTMLButton {
	return Button(title).ID(id).Attr("uk-toggle", fmt.Sprintf("target: #%s", id))
}

func Modal(id string, title string, body ...app.UI) app.HTMLDiv {
	var elems []app.UI
	elems = append(elems, app.H2().Class("uk-modal-title").Text(title))
	elems = append(elems, body...)
	elems = append(elems, app.P().Class("uk-text-right").Body(
		Button("Cancel").Class(ButtonDefaultClass, ModalCloseClass),
		Button("Save").Class(ButtonPrimaryClass),
	))
	return app.Div().ID(id).Attr("uk-modal", "").Body(
		app.Div().Class("uk-modal-dialog uk-modal-body").Body(elems...),
	)
}
