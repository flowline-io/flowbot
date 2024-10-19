package webhook

import "github.com/flowline-io/flowbot/internal/types"

type Rule struct {
	Id      string
	Secret  bool
	Handler func(ctx types.Context, content types.KV) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) ProcessRule(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	for _, rule := range r {
		result := rule.Handler(ctx, content)
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}
