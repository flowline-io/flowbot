package share

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/parser"
)

var commandRules = []command.Rule{
	{
		Define: "info",
		Help:   `Bot info`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return nil
		},
	},
	{
		Define: "input",
		Help:   `submit share`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.StoreForm(ctx, types.FormMsg{
				ID:    inputFormID,
				Title: "Share Content",
				Field: []types.FormField{
					{
						Key:         "content",
						Type:        types.FormFieldTextarea,
						ValueType:   types.FormFieldValueString,
						Value:       "",
						Label:       "Content",
						Placeholder: "Input content",
					},
				},
			})
		},
	},
	{
		Define: "share [string]",
		Help:   `Share text`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			text, _ := tokens[1].Value.String()
			return bots.StorePage(ctx, model.PageShare, text, types.TextMsg{Text: text})
		},
	},
}
