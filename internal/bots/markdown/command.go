package markdown

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/parser"
	"time"
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
		Define: "editor",
		Help:   `Bot info`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			p := types.KV{}
			p["uid"] = ctx.AsUser.UserId()
			flag, err := bots.StoreParameter(p, time.Now().Add(time.Hour))
			if err != nil {
				return types.TextMsg{Text: "error parameter"}
			}
			return types.LinkMsg{
				Title: "Markdown Editor",
				Url:   bots.ServiceURL(ctx, Name, serviceVersion, fmt.Sprintf("/editor/%s", flag), nil),
			}
		},
	},
}
