package collect

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRule_ID(t *testing.T) {
	r := Rule{Id: "test_collect"}
	assert.Equal(t, "test_collect", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	r := Rule{Id: "test_collect"}
	assert.Equal(t, types.CollectRule, r.TYPE())
}

func TestRuleset_ProcessAgent_MatchingRule(t *testing.T) {
	called := false
	rules := Ruleset{
		{
			Id:   "rule1",
			Help: "test rule",
			Args: []string{"arg1"},
			Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
				called = true
				v, _ := content.String("key")
				return types.TextMsg{Text: v}
			},
		},
	}

	ctx := types.Context{
		CollectRuleId: "rule1",
		AgentVersion:  1,
	}
	content := types.KV{"key": "hello"}
	result, err := rules.ProcessAgent(ctx, content)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, types.TextMsg{Text: "hello"}, result)
}

func TestRuleset_ProcessAgent_NoMatchingRule(t *testing.T) {
	rules := Ruleset{
		{
			Id: "rule1",
			Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
				return types.TextMsg{Text: "should not be called"}
			},
		},
	}

	ctx := types.Context{
		CollectRuleId: "nonexistent",
		AgentVersion:  1,
	}
	result, err := rules.ProcessAgent(ctx, types.KV{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRuleset_ProcessAgent_AgentVersionTooLow(t *testing.T) {
	rules := Ruleset{
		{
			Id: "rule1",
			Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
				return types.TextMsg{Text: "should not be called"}
			},
		},
	}

	ctx := types.Context{
		CollectRuleId: "rule1",
		AgentVersion:  0, // lower than ApiVersion
	}
	result, err := rules.ProcessAgent(ctx, types.KV{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent version too low")
	assert.Nil(t, result)
}

func TestRuleset_ProcessAgent_MultipleRules(t *testing.T) {
	callCount := 0
	rules := Ruleset{
		{
			Id: "rule1",
			Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
				callCount++
				return types.TextMsg{Text: "first"}
			},
		},
		{
			Id: "rule2",
			Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
				callCount++
				return types.TextMsg{Text: "second"}
			},
		},
	}

	ctx := types.Context{
		CollectRuleId: "rule2",
		AgentVersion:  1,
	}
	result, err := rules.ProcessAgent(ctx, types.KV{})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, types.TextMsg{Text: "second"}, result)
}

func TestRuleset_ProcessAgent_EmptyRuleset(t *testing.T) {
	rules := Ruleset{}
	ctx := types.Context{
		CollectRuleId: "rule1",
		AgentVersion:  1,
	}
	result, err := rules.ProcessAgent(ctx, types.KV{})
	require.NoError(t, err)
	assert.Nil(t, result)
}
