package share

import (
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/parser"
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
