package webhook

import "github.com/flowline-io/flowbot/pkg/types"

type Rule struct {
	Id      string
	Secret  bool
	Handler func(ctx types.Context, data []byte) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) ProcessRule(ctx types.Context, data []byte) (types.MsgPayload, error) {
	for _, rule := range r {
		result := rule.Handler(ctx, data)
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}
