package okr

import (
	_ "embed"
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/page/library"
	"github.com/flowline-io/flowbot/pkg/page/uikit"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const (
	okrPageId = "okr"
)

//go:embed static/okr.css
var exampleCss string

//go:embed static/okr.js
var exampleJs string

var pageRules = []page.Rule{
	{
		Id: okrPageId,
		UI: func(ctx types.Context, flag string) (*types.UI, error) {
			p, err := store.Database.ParameterGet(flag)
			if err != nil {
				return nil, err
			}

			sequence, _ := types.KV(p.Params).Int64("sequence")
			objective, err := store.Database.GetObjectiveBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				return nil, err
			}

			keyResult, err := store.Database.ListKeyResultsByObjectiveId(objective.ID)
			if err != nil {
				return nil, err
			}

			ratio := float64(0)
			if objective.TotalValue != 0 {
				ratio = float64(objective.CurrentValue) / float64(objective.TotalValue) * 100
			}

			css := []app.UI{
				uikit.Css(exampleCss),
			}
			js := []app.HTMLScript{
				uikit.Script(library.VueJs),
				uikit.Js(exampleJs),
			}

			app := uikit.App(
				uikit.Div(
					uikit.Text("Title").Class("okr-title"),
					uikit.Div(
						uikit.Text(objective.Title),
						uikit.Text("Progress").Class("okr-item-title"),
						uikit.Div(
							uikit.Div(
								uikit.Div().Class("progress-line progress-inner").Style("width", fmt.Sprintf("%.2f%%", ratio)),
							).Class("progress-bg-line progress-bg"),
							uikit.Text(ratio).Class("ratio"),
						).Class("okr-progress"),

						uikit.Text("Memo").Class("okr-item-title"),
						uikit.Text(objective.Memo).Class("okr-memo"),

						uikit.Text("Motive").Class("okr-item-title"),
						uikit.Text(objective.Motive).Class("okr-motive"),

						uikit.Text("Feasibility").Class("okr-item-title"),
						uikit.Text(objective.Feasibility).Class("okr-feasibility"),

						uikit.Text("KeyResult").Class("okr-item-title"),
						uikit.Div(
							app.Range(keyResult).Slice(func(i int) app.UI {
								return uikit.Div(
									uikit.Text(fmt.Sprintf("#%d %s", keyResult[i].Sequence, keyResult[i].Title)).Class("title"),
									uikit.Text(fmt.Sprintf("%d -> %d", keyResult[i].CurrentValue, keyResult[i].TargetValue)).Class("value"),
								).Class("okr-keyresult-item")
							}),
						).Class("okr-keyresult"),
					),
				).Class("okr"),
				uikit.Countdown(p.ExpiredAt),
			)

			return &types.UI{
				Title:  "OKR",
				App:    app,
				CSS:    css,
				JS:     js,
				Global: types.KV(p.Params),
			}, nil
		},
	},
}
