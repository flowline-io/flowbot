package search

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/parser"
)

var commandRules = []command.Rule{
	{
		Define: "search example",
		Help:   `Example command`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return nil
		},
	},
}
