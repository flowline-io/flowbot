package event

import (
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRule_ID(t *testing.T) {
	r := Rule{Id: "test_event"}
	assert.Equal(t, "test_event", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	r := Rule{Id: "test_event"}
	assert.Equal(t, types.EventRule, r.TYPE())
}

func TestRuleset_ProcessEvent_MatchingRule(t *testing.T) {
	called := false
	rules := Ruleset{
		{
			Id: "event1",
			Handler: func(ctx types.Context, param types.KV) error {
				called = true
				return nil
			},
		},
	}

	ctx := types.Context{EventRuleId: "event1"}
	err := rules.ProcessEvent(ctx, types.KV{"data": "value"})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRuleset_ProcessEvent_NoMatchingRule(t *testing.T) {
	rules := Ruleset{
		{
			Id: "event1",
			Handler: func(ctx types.Context, param types.KV) error {
				return errors.New("should not be called")
			},
		},
	}

	ctx := types.Context{EventRuleId: "nonexistent"}
	err := rules.ProcessEvent(ctx, types.KV{})
	require.NoError(t, err)
}

func TestRuleset_ProcessEvent_HandlerReturnsError(t *testing.T) {
	expectedErr := errors.New("handler error")
	rules := Ruleset{
		{
			Id: "event1",
			Handler: func(ctx types.Context, param types.KV) error {
				return expectedErr
			},
		},
	}

	ctx := types.Context{EventRuleId: "event1"}
	err := rules.ProcessEvent(ctx, types.KV{})
	assert.ErrorIs(t, err, expectedErr)
}

func TestRuleset_ProcessEvent_MultipleRulesStopsOnError(t *testing.T) {
	callOrder := []string{}
	rules := Ruleset{
		{
			Id: "event1",
			Handler: func(ctx types.Context, param types.KV) error {
				callOrder = append(callOrder, "first")
				return errors.New("first error")
			},
		},
		{
			Id: "event1", // duplicate ID
			Handler: func(ctx types.Context, param types.KV) error {
				callOrder = append(callOrder, "second")
				return nil
			},
		},
	}

	ctx := types.Context{EventRuleId: "event1"}
	err := rules.ProcessEvent(ctx, types.KV{})
	assert.Error(t, err)
	assert.Equal(t, []string{"first"}, callOrder)
}

func TestRuleset_ProcessEvent_EmptyRuleset(t *testing.T) {
	rules := Ruleset{}
	ctx := types.Context{EventRuleId: "event1"}
	err := rules.ProcessEvent(ctx, types.KV{})
	require.NoError(t, err)
}

func TestRuleset_ProcessEvent_PassesParams(t *testing.T) {
	var receivedParam types.KV
	rules := Ruleset{
		{
			Id: "event1",
			Handler: func(ctx types.Context, param types.KV) error {
				receivedParam = param
				return nil
			},
		},
	}

	ctx := types.Context{EventRuleId: "event1"}
	param := types.KV{"key1": "value1", "key2": int64(42)}
	err := rules.ProcessEvent(ctx, param)
	require.NoError(t, err)
	v, ok := receivedParam.String("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", v)
}
