package components

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type KeyboardHandler struct {
	app.Compo
}

func (h *KeyboardHandler) OnMount(ctx app.Context) {
	ctx.Handle("keydown", h.handleKeyDown)
}

func (h *KeyboardHandler) handleKeyDown(ctx app.Context, a app.Action) {
	key, ok := a.Value.(string)
	if !ok {
		return
	}

	switch key {
	case "/":
		ctx.NewAction("focus-search", nil)
		ShowToast(ctx, "Search focused", "info")
	case "?":
		h.showHelp(ctx)
	}
}

func (h *KeyboardHandler) showHelp(ctx app.Context) {
	ShowToast(ctx, "Shortcuts: / = search, ? = help", "info")
}

func (h *KeyboardHandler) Render() app.UI {
	return app.Div()
}
