package langchain

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/tmc/langchaingo/llms"
)

type Rule struct {
	Id      string
	Tool    llms.Tool
	Execute func(ctx types.Context, args types.KV) (string, error)
}

type Ruleset []Rule

func (r Ruleset) ProcessRule(ctx types.Context, args types.KV) (string, error) {
	for _, item := range r {
		if item.Id == ctx.ToolRuleId {
			return item.Execute(ctx, args)
		}
	}
	return "", nil
}
