package bookmark

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
)

var commandRules = []command.Rule{
	{
		Define: "bookmark list",
		Help:   `newest 10`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(hoarder.ID, hoarder.EndpointKey)
			apiKey, _ := providers.GetConfig(hoarder.ID, hoarder.ApikeyKey)
			client := hoarder.NewHoarder(endpoint.String(), apiKey.String())
			resp, err := client.GetAllBookmarks(10)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			var header []string
			var row [][]interface{}
			if len(resp.Bookmarks) > 0 {
				header = []string{"Id", "Url", "Title", "TaggingStatus"}
				for _, v := range resp.Bookmarks {
					row = append(row, []interface{}{v.Id, v.Content.Url, v.Title, v.TaggingStatus})
				}
			}

			return bots.StorePage(ctx, model.PageTable, "Newest Bookmark List", types.TableMsg{
				Title:  "Newest Bookmark List",
				Header: header,
				Row:    row,
			})
		},
	},
}
