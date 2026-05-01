package gitea

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "gitea",
		Help:   `Example command`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			client, err := gitea.GetClient()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			resp, err := client.GetRepositories("demo", "example")
			_, _ = fmt.Println(resp, err)

			return types.TextMsg{Text: "ok"}
		},
	},
}
