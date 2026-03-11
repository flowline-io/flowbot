package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

type LoadingOverlay struct {
	app.Compo

	Show    bool
	Message string
}

func (l *LoadingOverlay) Render() app.UI {
	if !l.Show {
		return app.Div()
	}

	message := l.Message
	if message == "" {
		message = "Loading..."
	}

	return app.Div().Class("fixed inset-0 z-[9998] flex items-center justify-center bg-base-300/80 backdrop-blur-sm").Body(
		app.Div().Class("flex flex-col items-center gap-4").Body(
			app.Span().Class("loading loading-spinner loading-lg text-primary"),
			app.Span().Class("text-base-content font-medium").Text(message),
		),
	)
}

func ShowLoading(ctx app.Context, message string) {
	ctx.NewAction("show-loading", app.T("message", message))
}

func HideLoading(ctx app.Context) {
	ctx.NewAction("hide-loading")
}

func (l *LoadingOverlay) OnMount(ctx app.Context) {
	ctx.Handle("show-loading", func(ctx app.Context, a app.Action) {
		msg := a.Tags.Get("message")
		ctx.Dispatch(func(ctx app.Context) {
			l.Show = true
			l.Message = msg
		})
	})

	ctx.Handle("hide-loading", func(ctx app.Context, a app.Action) {
		ctx.Dispatch(func(ctx app.Context) {
			l.Show = false
			l.Message = ""
		})
	})
}
