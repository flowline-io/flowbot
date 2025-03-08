package bookmark

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/hoarder"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "bookmark list",
		Help:   `newest 10`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			client := hoarder.GetClient()
			bookmarks, err := client.GetAllBookmarks(10)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			var header []string
			var row [][]interface{}
			if len(bookmarks) > 0 {
				header = []string{"Id", "Title", "TaggingStatus"}
				for _, v := range bookmarks {
					row = append(row, []interface{}{v.Id, v.Content.Title, v.TaggingStatus})
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
