package webhook

import "github.com/flowline-io/flowbot/pkg/types"

type Rule struct {
	Id      string
	Secret  bool
	Handler func(ctx types.Context, data []byte) types.MsgPayload
}

func (r Rule) ID() string {
	return r.Id
}

func (r Rule) TYPE() types.RulesetType {
	return types.WebhookRule
}

type Ruleset []Rule

func (r Ruleset) ProcessRule(ctx types.Context, data []byte) (types.MsgPayload, error) {
	var result types.MsgPayload
	for _, rule := range r {
		if rule.Id == ctx.WebhookRuleId {
			result = rule.Handler(ctx, data)
		}
	}
	return result, nil
}
