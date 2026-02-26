package bookmark

import (
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/chatbot"
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
			resp, err := client.GetAllBookmarks(&hoarder.BookmarksQuery{Limit: 10})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			var header []string
			var row [][]any
			if resp != nil && len(resp.Bookmarks) > 0 {
				header = []string{"Id", "Title", "TaggingStatus"}
				for _, v := range resp.Bookmarks {
					row = append(row, []any{v.Id, v.Content.Title, v.TaggingStatus})
				}
			}

			return chatbot.StorePage(ctx, model.PageTable, "Newest Bookmark List", types.TableMsg{
				Title:  "Newest Bookmark List",
				Header: header,
				Row:    row,
			})
		},
	},
}
