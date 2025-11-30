package flows

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "flows ui",
		Help:   `flows ui`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			url, err := chatbot.PageURL(ctx, flowsListPageId, nil, time.Hour)
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error %v", err)}
			}
			return types.TextMsg{Text: url}
		},
	},
}
