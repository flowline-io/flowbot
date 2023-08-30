package clipboard

import (
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store/model"
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
		Define: "share [string]",
		Help:   `share clipboard to linkit`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			txt, _ := tokens[1].Value.String()
			data := types.KV{}
			data["txt"] = txt
			return bots.StoreInstruct(ctx, types.InstructMsg{
				No:       types.Id(),
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
