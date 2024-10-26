package gitea

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
)

var commandRules = []command.Rule{
	{
		Define: "gitea",
		Help:   `Example command`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(gitea.ID, gitea.EndpointKey)
			token, _ := providers.GetConfig(gitea.ID, gitea.TokenKey)
			client, err := gitea.NewGitea(endpoint.String(), token.String())
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			resp, err := client.GetRepositories("demo", "example")
			_, _ = fmt.Println(resp, err)

			return nil
		},
	},
}
