package event

import (
	"github.com/flowline-io/flowbot/pkg/types"
)

type Rule struct {
	Id      string
	Handler func(ctx types.Context, param types.KV) error
}

type Ruleset []Rule

func (r Ruleset) ProcessEvent(ctx types.Context, param types.KV) (err error) {
	for _, rule := range r {
		if rule.Id == ctx.EventId {
			err = rule.Handler(ctx, param)
			if err != nil {
				return
			}
		}
	}
	return
}
