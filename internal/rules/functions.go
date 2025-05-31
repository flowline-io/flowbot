package rules

import (
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/action"
)

func a() {
	action.Functions.Register("sendMsg", func(ctx types.RuleContext, msg types.RuleMsg) {
		ctx.TellSuccess(msg)
		ctx.TellNext(msg, types.True)
		return
	})
}
