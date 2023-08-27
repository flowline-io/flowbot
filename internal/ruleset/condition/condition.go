package condition

import "github.com/sysatom/flowbot/internal/types"

type Rule struct {
	Condition string
	Handler   func(ctx types.Context, forwarded types.MsgPayload) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) ProcessCondition(ctx types.Context, forwarded types.MsgPayload) (types.MsgPayload, error) {
	var result types.MsgPayload
	for _, rule := range r {
		if rule.Condition == ctx.Condition {
			result = rule.Handler(ctx, forwarded)
		}
	}
	return result, nil
}
