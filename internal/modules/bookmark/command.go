// Package bookmark implements the bookmark management module.
package bookmark

import (
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "bookmark list",
		Help:   `newest 10`,
		Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
			res, err := ability.Invoke(ctx.Context(), hub.CapBookmark, ability.OpBookmarkList, map[string]any{"limit": 10})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			var header []string
			var row [][]any
			bookmarks, ok := res.Data.([]*ability.Bookmark)
			if !ok {
				bookmarks = nil
			}
			if len(bookmarks) > 0 {
				header = []string{"Id", "Title", "URL"}
				for _, v := range bookmarks {
					row = append(row, []any{v.ID, v.Title, v.URL})
				}
			}

			return module.StorePage(ctx, schema.PageTable, "Newest Bookmark List", types.TableMsg{
				Title:  "Newest Bookmark List",
				Header: header,
				Row:    row,
			})
		},
	},
}
