package search

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "search",
		Help:   `Search page`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			url, err := chatbot.PageURL(ctx, searchPageId, nil, 24*time.Hour)
			if err != nil {
				return types.TextMsg{Text: "error"}
			}

			return types.LinkMsg{Url: url}
		},
	},
}
