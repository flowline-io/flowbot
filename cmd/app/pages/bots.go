package pages

import (
	"fmt"

	"github.com/flowline-io/flowbot/cmd/app/api"
	"github.com/flowline-io/flowbot/cmd/app/components"
	"github.com/flowline-io/flowbot/cmd/app/state"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Bots struct {
	app.Compo

	bots  []admin.BotInfo
	total int64

	search  string
	loading bool
}

func (b *Bots) OnNav(ctx app.Context) {
	if !state.IsAuthenticated(ctx) {
		ctx.Navigate("/admin/login")
		return
	}

	b.loadBots(ctx)
}

func (b *Bots) loadBots(ctx app.Context) {
	b.loading = true
	token := state.Token(ctx)

	ctx.Async(func() {
		resp, err := api.ListBots(token)
		ctx.Dispatch(func(ctx app.Context) {
			b.loading = false
			if err != nil {
				components.ShowToast(ctx, "Failed to load bots: "+err.Error(), "error")
				return
			}
			b.bots = resp.Items
			b.total = resp.Total
		})
	})
}

func (b *Bots) Render() app.UI {
	return components.WithLayout(
		app.Div().Class("flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8").Body(
			app.Div().Body(
				app.H1().Class("text-3xl font-bold tracking-tight").Text("Bot Modules"),
				app.P().Class("text-base-content/50 mt-1").
					Text(fmt.Sprintf("%d modules registered", b.total)),
			),
			app.Div().Class("flex gap-2").Body(
				app.Div().Class("relative").Body(
					app.Div().Class("absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none text-base-content/40").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>`),
					),
					app.Input().
						Type("text").
						Class("input input-bordered input-md w-full pl-10 pr-4 bg-base-200/30 focus:bg-base-100 transition-colors duration-200").
						Placeholder("Search bots...").
						Value(b.search).
						OnChange(b.handleSearch),
				),
			),
		),

		app.If(b.loading, func() app.UI {
			return app.Div().Class("flex justify-center py-16").Body(
				app.Span().Class("loading loading-spinner loading-lg text-primary"),
			)
		}).Else(func() app.UI {
			if len(b.bots) == 0 {
				return app.Div().Class("card bg-base-100/80 backdrop-blur-sm shadow-xl border border-base-200/50").Body(
					app.Div().Class("card-body items-center py-16 text-center").Body(
						app.Div().Class("w-20 h-20 rounded-full bg-base-200/50 flex items-center justify-center mb-4").Body(
							app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-base-content/30" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>`),
						),
						app.P().Class("text-base-content/50 font-medium text-lg").Text("No bot modules registered"),
						app.P().Class("text-base-content/30 text-sm mt-1").Text("Bot modules will appear here when registered"),
					),
				)
			}
			return app.Div().Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5").Body(
				b.renderBotCards()...,
			)
		}),
	)
}

func (b *Bots) renderBotCards() []app.UI {
	cards := make([]app.UI, 0, len(b.bots))
	for _, bot := range b.bots {
		bt := bot
		cards = append(cards, b.botCard(bt))
	}
	return cards
}

func (b *Bots) botCard(bot admin.BotInfo) app.UI {
	features := make([]app.UI, 0)
	if bot.HasForm {
		features = append(features, app.Span().Class("badge badge-primary badge-sm").Text("Form"))
	}
	if bot.HasCron {
		features = append(features, app.Span().Class("badge badge-secondary badge-sm").Text("Cron"))
	}
	if bot.HasWebhook {
		features = append(features, app.Span().Class("badge badge-accent badge-sm").Text("Webhook"))
	}
	if len(bot.Commands) > 0 {
		features = append(features, app.Span().Class("badge badge-ghost badge-sm").Text(fmt.Sprintf("%d cmds", len(bot.Commands))))
	}

	statusBadge := app.Span().Class("badge badge-success badge-sm gap-1").Body(
		app.Span().Class("w-1.5 h-1.5 rounded-full bg-success animate-pulse"),
		app.Text("Active"),
	)
	if !bot.Enabled {
		statusBadge = app.Span().Class("badge badge-error badge-sm gap-1").Body(
			app.Span().Class("w-1.5 h-1.5 rounded-full bg-error"),
			app.Text("Disabled"),
		)
	}

	return app.Div().Class("card bg-base-100/80 backdrop-blur-sm shadow-lg border border-base-200/50 hover:shadow-xl hover:border-primary/20 transition-all duration-300 hover:-translate-y-1").Body(
		app.Div().Class("card-body p-5").Body(
			app.Div().Class("flex items-center justify-between mb-4").Body(
				app.Div().Class("flex items-center gap-3").Body(
					app.Div().Class("w-10 h-10 rounded-xl bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>`),
					),
					app.Span().Class("text-lg font-bold").Body(
						components.HighlightText(bot.Name, b.search),
					),
				),
				statusBadge,
			),
			app.If(len(features) > 0, func() app.UI {
				return app.Div().Class("flex flex-wrap gap-2 mb-3").Body(features...)
			}),
			app.P().Class("text-sm text-base-content/60 leading-relaxed").Body(
				components.HighlightText(bot.Description, b.search),
			),
		),
	)
}

func (b *Bots) handleSearch(ctx app.Context, e app.Event) {
	b.search = ctx.JSSrc().Get("value").String()
	b.loadBots(ctx)
}
