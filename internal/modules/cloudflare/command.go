package cloudflare

import (
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "cloudflare setting",
		Help:   `cloudflare setting`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return module.SettingMsg(ctx, Name)
		},
	},
}
