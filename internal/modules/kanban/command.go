package kanban

import (
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "kanban status",
		Help:   `Show kanban status`,
		Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
			return types.EmptyMsg{}
		},
	},
}
