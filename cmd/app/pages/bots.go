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
		app.Div().Class("flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6").Body(
			app.Div().Body(
				app.H1().Class("text-3xl font-bold tracking-tight").Text("Bot Modules"),
				app.P().Class("text-base-content/50 mt-1").Text(
					fmt.Sprintf("%d modules registered", b.total),
				),
			),
			app.Div().Class("flex gap-2").Body(
				app.Div().Class("relative max-w-xs").Body(
					app.Div().Class("absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none text-base-content/40").Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>`),
					),
					app.Input().
						Type("text").
						Class("input input-bordered input-sm w-full pl-9").
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
				return app.Div().Class("text-center py-16").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-4 text-base-content/30" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>`),
					app.P().Class("text-base-content/50").Text("No bot modules registered"),
				)
			}
			return app.Div().Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4").Body(
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
		features = append(features, app.Span().Class("badge badge-xs badge-primary").Text("Form"))
	}
	if bot.HasCron {
		features = append(features, app.Span().Class("badge badge-xs badge-secondary").Text("Cron"))
	}
	if bot.HasWebhook {
		features = append(features, app.Span().Class("badge badge-xs badge-accent").Text("Webhook"))
	}
	if len(bot.Commands) > 0 {
		features = append(features, app.Span().Class("badge badge-xs badge-ghost").Text(fmt.Sprintf("%d cmds", len(bot.Commands))))
	}

	statusBadge := app.Span().Class("badge badge-success badge-sm").Text("Active")
	if !bot.Enabled {
		statusBadge = app.Span().Class("badge badge-error badge-sm").Text("Disabled")
	}

	return app.Div().Class("card bg-base-100 shadow-md hover:shadow-lg transition-shadow").Body(
		app.Div().Class("card-body p-4").Body(
			app.Div().Class("flex items-center justify-between mb-3").Body(
				app.Div().Class("flex items-center gap-2").Body(
					app.Span().Class("text-lg font-bold").Body(
						components.HighlightText(bot.Name, b.search),
					),
					statusBadge,
				),
			),
			app.If(len(features) > 0, func() app.UI {
				return app.Div().Class("flex flex-wrap gap-1 mb-2").Body(features...)
			}),
			app.P().Class("text-sm text-base-content/60").Body(
				components.HighlightText(bot.Description, b.search),
			),
		),
	)
}

func (b *Bots) handleSearch(ctx app.Context, e app.Event) {
	b.search = ctx.JSSrc().Get("value").String()
	b.loadBots(ctx)
}
