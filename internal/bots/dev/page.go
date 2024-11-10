package dev

import (
	_ "embed"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/page/library"
	"github.com/flowline-io/flowbot/pkg/page/uikit"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const (
	devPageId  = "dev"
	jsonPageId = "json"
)

//go:embed static/example.css
var exampleCss string

//go:embed static/example.js
var exampleJs string

//go:embed static/json.js
var jsonJs string

var pageRules = []page.Rule{
	{
		Id: devPageId,
		UI: func(ctx types.Context, flag string) (*types.UI, error) {
			p, err := store.Database.ParameterGet(flag)
			if err != nil {
				return nil, err
			}

			css := []app.UI{
				uikit.Style(library.GithubMarkdownCss),
				uikit.Css(exampleCss),
			}
			js := []app.HTMLScript{
				uikit.Script(library.VueJs),
				uikit.Script(library.AxiosJs),
				uikit.Script(library.JoiJs),
				uikit.Js(exampleJs),
			}

			app := uikit.App(
				uikit.H1("{{ message }}").Class(uikit.TextCenterClass),
				uikit.Grid(
					uikit.Card("One", app.Div().Text("Lorem ipsum dolor sit amet, consectetur adipiscing elit")),
					uikit.Card("Two", app.Div().Text("Lorem ipsum dolor sit amet, consectetur adipiscing elit")),
				).Class(uikit.FlexClass, uikit.FlexCenterClass),
				uikit.Icon("home"),
				uikit.Div(
					uikit.Label("One"),
					uikit.Label("Two").Class(uikit.LabelSuccessClass),
					uikit.Label("Three").Class(uikit.LabelWarningClass),
					uikit.Label("Four").Class(uikit.LabelDangerClass),
				),
				uikit.Article("title", time.Now().Format(time.DateTime), uikit.Text("article......")),
				uikit.DividerIcon(),
				uikit.Form(
					uikit.Margin(
						uikit.FormLabel("text", "f1"),
						uikit.FormControls(
							uikit.Input().Name("text").ID("f1").Attr("v-model", "form.text"),
						),
					),
					uikit.Margin(
						uikit.FormLabel("select", "f2"),
						uikit.FormControls(
							uikit.Select(
								uikit.Option("Option 01"),
								uikit.Option("Option 02"),
							).Name("select").ID("f2").Attr("v-model", "form.select"),
						),
					),
					uikit.Margin(
						uikit.FormLabel("radio", "f3"),
						uikit.FormControls(
							app.Label().Body(uikit.Radio().Name("radio").Attr("v-model", "form.radio"), uikit.Text("option 1").Class(uikit.InlineClass)),
							app.Label().Body(uikit.Radio().Name("radio").Attr("v-model", "form.radio"), uikit.Text("option 2").Class(uikit.InlineClass)),
						),
					),
					uikit.Button("Submit").Type("button").Attr("@click", "submit"),
				).Method(http.MethodPost).Action("/service/dev/example"),
				uikit.Placeholder("Lorem ipsum dolor sit amet, consectetur adipiscing elit."),
				uikit.Progress(10, 100),
				uikit.Button("click event").Attr("@click", "greet"),
				uikit.Table(
					uikit.THead(
						uikit.Tr(
							uikit.Th(uikit.Text("heading")),
							uikit.Th(uikit.Text("heading")),
							uikit.Th(uikit.Text("heading")),
						)),
					uikit.TBody(
						uikit.Tr(
							uikit.Td(uikit.Text("data")),
							uikit.Td(uikit.Text("data")),
							uikit.Td(uikit.Text("data")),
						),
					),
					uikit.TFoot(
						uikit.Tr(
							uikit.Td(uikit.Text("footer")),
							uikit.Td(uikit.Text("footer")),
							uikit.Td(uikit.Text("footer")),
						),
					),
				).Class(uikit.TableDividerClass, uikit.TableHoverClass),
				uikit.ModalToggle("example_modal", "modal"),
				uikit.Modal("example_modal", "modal", uikit.Text("content......")),
				uikit.Image("https://images.unsplash.com/photo-1490822180406-880c226c150b?fit=crop&w=650&h=433&q=80"),
				uikit.Countdown(p.ExpiredAt),
			)

			return &types.UI{
				App:    app,
				CSS:    css,
				JS:     js,
				Global: types.KV(p.Params),
			}, nil
		},
	},
	{
		Id: jsonPageId,
		UI: func(ctx types.Context, flag string) (*types.UI, error) {
			css := []app.UI{
				uikit.Style(library.JsonFormatterCss),
			}
			js := []app.HTMLScript{
				uikit.Script(library.JsonFormatterJs),
				uikit.Js(jsonJs),
			}

			app := uikit.App(
				uikit.H1("JSON Formatter").Class(uikit.TextCenterClass),
				uikit.Textarea().ID("data"),
				uikit.Div().ID("view"),
			)

			return &types.UI{
				App: app,
				CSS: css,
				JS:  js,
			}, nil
		},
	},
}
