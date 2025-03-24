package tool

import (
	"fmt"

	llmTool "github.com/cloudwego/eino/components/tool"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Rule func(ctx types.Context) (llmTool.InvokableTool, error)

func (r Rule) ID() string {
	return ""
}

func (r Rule) TYPE() types.RulesetType {
	return types.ToolRule
}

type Ruleset []Rule

func (r Ruleset) ProcessRule(ctx types.Context, argumentsInJSON string) (string, error) {
	for _, item := range r {
		t, err := item(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to create tool: %w", err)
		}
		i, err := t.Info(ctx.Context())
		if err != nil {
			return "", fmt.Errorf("failed to get tool info: %w", err)
		}
		if i.Name == ctx.ToolRuleId {
			return t.InvokableRun(ctx.Context(), argumentsInJSON)
		}
	}
	return "", nil
}
