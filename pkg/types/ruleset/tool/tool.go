package tool

import (
	"fmt"
	llmTool "github.com/cloudwego/eino/components/tool"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Rule struct {
	Id   string
	Tool func(ctx types.Context) (llmTool.InvokableTool, error)
}

type Ruleset []Rule

func (r Ruleset) ProcessRule(ctx types.Context, argumentsInJSON string) (string, error) {
	for _, item := range r {
		if item.Id == ctx.ToolRuleId {
			t, err := item.Tool(ctx)
			if err != nil {
				return "", fmt.Errorf("failed to create tool: %w", err)
			}
			return t.InvokableRun(ctx.Context(), argumentsInJSON)
		}
	}
	return "", nil
}
