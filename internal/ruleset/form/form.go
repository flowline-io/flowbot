package form

import "github.com/sysatom/flowbot/internal/types"

type Rule struct {
	Id         string
	IsLongTerm bool
	Title      string
	Field      []types.FormField
	Handler    func(ctx types.Context, values types.KV) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) ProcessForm(ctx types.Context, values types.KV) (types.MsgPayload, error) {
	var result types.MsgPayload
	for _, rule := range r {
		if rule.Id == ctx.FormRuleId {
			result = rule.Handler(ctx, values)
		}
	}
	return result, nil
}
