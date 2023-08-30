package workflow

import "github.com/flowline-io/flowbot/internal/types"

type Rule struct {
	Id           string
	Title        string
	Desc         string
	InputSchema  []types.FormField
	OutputSchema []types.FormField
	Run          func(ctx types.Context, input types.KV) (types.KV, error)
}

type Ruleset []Rule

func (r Ruleset) ProcessRule(ctx types.Context, input types.KV) (types.KV, error) {
	for _, rule := range r {
		if rule.Id == ctx.WorkflowRuleId {
			return rule.Run(ctx, input)
		}
	}
	return types.KV{}, nil
}
