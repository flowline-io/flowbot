package tool

import (
	"testing"

	llmTool "github.com/cloudwego/eino/components/tool"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRule_ID(t *testing.T) {
	var r Rule = func(ctx types.Context) (llmTool.InvokableTool, error) {
		return nil, nil
	}
	// Rule.ID() always returns empty string
	assert.Equal(t, "", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	var r Rule = func(ctx types.Context) (llmTool.InvokableTool, error) {
		return nil, nil
	}
	assert.Equal(t, types.ToolRule, r.TYPE())
}

func TestRuleset_Empty(t *testing.T) {
	rules := Ruleset{}
	assert.Len(t, rules, 0)
}

func TestRuleset_Creation(t *testing.T) {
	rules := Ruleset{
		func(ctx types.Context) (llmTool.InvokableTool, error) {
			return nil, nil
		},
	}
	assert.Len(t, rules, 1)
}
