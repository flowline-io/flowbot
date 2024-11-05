package torrent

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/parser"
)

var commandRules = []command.Rule{
	{
		Define: "torrent clear",
		Help:   `clear torrents`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := torrentClear(ctx.Context())
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
}
