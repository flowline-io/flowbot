package tool

import (
	"testing"

	"github.com/flowline-io/flowbot/internal/agents"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRule_ID(t *testing.T) {
	var r Rule = func(ctx types.Context) (agents.InvokableTool, error) {
		return nil, nil
	}
	assert.Equal(t, "", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	var r Rule = func(ctx types.Context) (agents.InvokableTool, error) {
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
		func(ctx types.Context) (agents.InvokableTool, error) {
			return nil, nil
		},
	}
	assert.Len(t, rules, 1)
}
