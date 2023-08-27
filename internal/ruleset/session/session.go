package session

import "github.com/sysatom/flowbot/internal/types"

type Rule struct {
	Id      string
	Title   string
	Handler func(ctx types.Context, content interface{}) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) ProcessSession(ctx types.Context, content interface{}) (types.MsgPayload, error) {
	var result types.MsgPayload
	for _, rule := range r {
		if rule.Id == ctx.SessionRuleId {
			result = rule.Handler(ctx, content)
		}
	}
	return result, nil
}
