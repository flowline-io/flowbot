package webhook

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRule_ID(t *testing.T) {
	r := Rule{Id: "test_webhook"}
	assert.Equal(t, "test_webhook", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	r := Rule{Id: "test_webhook"}
	assert.Equal(t, types.WebhookRule, r.TYPE())
}

func TestRuleset_ProcessRule_MatchingRule(t *testing.T) {
	called := false
	rules := Ruleset{
		{
			Id:     "webhook1",
			Secret: true,
			Handler: func(ctx types.Context, data []byte) types.MsgPayload {
				called = true
				return types.TextMsg{Text: string(data)}
			},
		},
	}

	ctx := types.Context{WebhookRuleId: "webhook1"}
	result, err := rules.ProcessRule(ctx, []byte("test data"))
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, types.TextMsg{Text: "test data"}, result)
}

func TestRuleset_ProcessRule_NoMatchingRule(t *testing.T) {
	rules := Ruleset{
		{
			Id: "webhook1",
			Handler: func(ctx types.Context, data []byte) types.MsgPayload {
				return types.TextMsg{Text: "should not be called"}
			},
		},
	}

	ctx := types.Context{WebhookRuleId: "nonexistent"}
	result, err := rules.ProcessRule(ctx, []byte("data"))
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRuleset_ProcessRule_MultipleRules(t *testing.T) {
	rules := Ruleset{
		{
			Id: "wh1",
			Handler: func(ctx types.Context, data []byte) types.MsgPayload {
				return types.TextMsg{Text: "first"}
			},
		},
		{
			Id: "wh2",
			Handler: func(ctx types.Context, data []byte) types.MsgPayload {
				return types.TextMsg{Text: "second"}
			},
		},
	}

	ctx := types.Context{WebhookRuleId: "wh2"}
	result, err := rules.ProcessRule(ctx, []byte{})
	require.NoError(t, err)
	assert.Equal(t, types.TextMsg{Text: "second"}, result)
}

func TestRuleset_ProcessRule_EmptyRuleset(t *testing.T) {
	rules := Ruleset{}
	ctx := types.Context{WebhookRuleId: "wh1"}
	result, err := rules.ProcessRule(ctx, []byte("data"))
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRuleset_ProcessRule_SecretFlag(t *testing.T) {
	rules := Ruleset{
		{Id: "public_hook", Secret: false},
		{Id: "secret_hook", Secret: true},
	}
	assert.False(t, rules[0].Secret)
	assert.True(t, rules[1].Secret)
}

func TestRuleset_ProcessRule_EmptyData(t *testing.T) {
	rules := Ruleset{
		{
			Id: "wh1",
			Handler: func(ctx types.Context, data []byte) types.MsgPayload {
				return types.TextMsg{Text: "received empty"}
			},
		},
	}

	ctx := types.Context{WebhookRuleId: "wh1"}
	result, err := rules.ProcessRule(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, types.TextMsg{Text: "received empty"}, result)
}

func TestRuleset_ProcessRule_LargePayload(t *testing.T) {
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	var receivedLen int
	rules := Ruleset{
		{
			Id: "wh1",
			Handler: func(ctx types.Context, data []byte) types.MsgPayload {
				receivedLen = len(data)
				return types.TextMsg{Text: "ok"}
			},
		},
	}

	ctx := types.Context{WebhookRuleId: "wh1"}
	result, err := rules.ProcessRule(ctx, largeData)
	require.NoError(t, err)
	assert.Equal(t, 1024*1024, receivedLen)
	assert.Equal(t, types.TextMsg{Text: "ok"}, result)
}
