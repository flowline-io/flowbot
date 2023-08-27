package action

import (
	"errors"
	"github.com/sysatom/flowbot/internal/types"
)

type Rule struct {
	Id         string
	IsLongTerm bool
	Title      string
	Option     []string
	Handler    map[string]func(ctx types.Context) types.MsgPayload
}

type Ruleset []Rule

func (r Ruleset) ProcessAction(ctx types.Context, option string) (types.MsgPayload, error) {
	var result types.MsgPayload
	for _, rule := range r {
		if rule.Id == ctx.ActionRuleId {
			if handler, ok := rule.Handler[option]; ok {
				result = handler(ctx)
			} else {
				return nil, errors.New("action rule: error option")
			}
		}
	}
	return result, nil
}
