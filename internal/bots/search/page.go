package search

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/page/uikit"
	"github.com/flowline-io/flowbot/pkg/providers/meilisearch"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/page"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"net/http"
)

const (
	searchPageId = "search"
)

var pageRules = []page.Rule{
	{
		Id: searchPageId,
		UI: func(ctx types.Context, flag string, args types.KV) (*types.UI, error) {
			keyword, _ := args.String("q")
			source, _ := args.String("source")

			// search
			if keyword == "" {
				return nil, fmt.Errorf("empty keyword")
			}

			list, _, err := meilisearch.NewMeiliSearch().Search(source, keyword, 1, 10)
			if err != nil {
				return nil, fmt.Errorf("search error: %s", err)
			}

			var items []app.UI
			for _, item := range list {
				items = append(items, uikit.Tr(
					uikit.Td(uikit.Label(item.Source)),
					uikit.Td(
						uikit.Link(item.Title, item.Url).Target("_blank"),
					),
					uikit.Td(uikit.Text(item.Description)),
				))
			}

			// UI
			app := uikit.App(
				uikit.H2("Search").Class(uikit.TextCenterClass),
				uikit.Form(
					uikit.Margin(
						uikit.FormControls(
							uikit.Input().Name("q").Value(keyword).Placeholder("Keyword"),
						),
					),
					uikit.Button("Search").Type("submit"),
				).Method(http.MethodGet).Action(""),
				uikit.Table(
					uikit.THead(
						uikit.Tr(
							uikit.Th(uikit.Text("source")),
							uikit.Th(uikit.Text("title")),
							uikit.Th(uikit.Text("description")),
						)),
					uikit.TBody(
						items...,
					),
				).Class(uikit.TableDividerClass, uikit.TableHoverClass),
			)

			return &types.UI{
				App: app,
			}, nil
		},
	},
}
