package event

import "github.com/sysatom/flowbot/internal/types"

type Rule struct {
	Event   types.GroupEvent
	Handler func(ctx types.Context, head types.KV, content interface{}) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) ProcessEvent(ctx types.Context, head types.KV, content interface{}) ([]types.MsgPayload, error) {
	var result []types.MsgPayload
	for _, rule := range r {
		if ctx.GroupEvent == rule.Event {
			result = append(result, rule.Handler(ctx, head, content))
		}
	}
	return result, nil
}
