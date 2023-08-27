package rust

import (
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/crates"
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
		Define: "crate [string]",
		Help:   `crate info`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			name, _ := tokens[1].Value.String()

			api := crates.NewCrates()
			resp, err := api.Info(name)
			if err != nil {
				logs.Err.Println("bot command crate[number]", err)
				return types.TextMsg{Text: "error create"}
			}
			if resp == nil || resp.Crate.ID == "" {
				return types.TextMsg{Text: "empty create"}
			}

			return types.CrateMsg{
				ID:            resp.Crate.ID,
				Name:          resp.Crate.Name,
				Description:   resp.Crate.Description,
				Documentation: resp.Crate.Documentation,
				Homepage:      resp.Crate.Homepage,
				Repository:    resp.Crate.Repository,
				NewestVersion: resp.Crate.NewestVersion,
				Downloads:     resp.Crate.Downloads,
			}
		},
	},
}
