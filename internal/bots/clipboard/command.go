package clipboard

import (
	"github.com/sysatom/flowbot/internal/bots"
	"github.com/sysatom/flowbot/internal/ruleset/command"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/parser"
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
		Define: "share [string]",
		Help:   `share clipboard to linkit`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			txt, _ := tokens[1].Value.String()
			data := types.KV{}
			data["txt"] = txt
			return bots.StoreInstruct(ctx, types.InstructMsg{
				No:       types.Id().String(),
				Object:   model.InstructObjectLinkit,
				Bot:      Name,
				Flag:     ShareInstruct,
				Content:  data,
				Priority: model.InstructPriorityDefault,
				State:    model.InstructCreate,
				ExpireAt: time.Now().Add(time.Hour),
			})
		},
	},
}