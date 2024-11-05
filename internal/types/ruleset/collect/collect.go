package collect

import (
	"errors"

	"github.com/flowline-io/flowbot/internal/types"
)

type Rule struct {
	Id      string
	Help    string
	Args    []string
	Handler func(ctx types.Context, content types.KV) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) ProcessAgent(ctx types.Context, content types.KV) (types.MsgPayload, error) {
	if types.ApiVersion > ctx.AgentVersion {
		return nil, errors.New("agent version too low")
	}
	var result types.MsgPayload
	for _, rule := range r {
		if rule.Id == ctx.CollectId {
			result = rule.Handler(ctx, content)
		}
	}
	return result, nil
}
