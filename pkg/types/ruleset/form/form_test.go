package form

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRule_ID(t *testing.T) {
	r := Rule{Id: "test_form"}
	assert.Equal(t, "test_form", r.ID())
}

func TestRule_TYPE(t *testing.T) {
	r := Rule{Id: "test_form"}
	assert.Equal(t, types.FormRule, r.TYPE())
}

func TestRuleset_ProcessForm_MatchingRule(t *testing.T) {
	called := false
	rules := Ruleset{
		{
			Id:    "form1",
			Title: "Test Form",
			Field: []types.FormField{
				{Type: types.FormFieldText, Key: "name", Label: "Name"},
			},
			Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
				called = true
				name, _ := values.String("name")
				return types.TextMsg{Text: "Hello " + name}
			},
		},
	}

	ctx := types.Context{FormRuleId: "form1"}
	result, err := rules.ProcessForm(ctx, types.KV{"name": "World"})
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, types.TextMsg{Text: "Hello World"}, result)
}

func TestRuleset_ProcessForm_NoMatchingRule(t *testing.T) {
	rules := Ruleset{
		{
			Id: "form1",
			Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
				return types.TextMsg{Text: "should not be called"}
			},
		},
	}

	ctx := types.Context{FormRuleId: "nonexistent"}
	result, err := rules.ProcessForm(ctx, types.KV{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRuleset_ProcessForm_MultipleRules(t *testing.T) {
	rules := Ruleset{
		{
			Id: "form1",
			Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
				return types.TextMsg{Text: "first"}
			},
		},
		{
			Id: "form2",
			Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
				return types.TextMsg{Text: "second"}
			},
		},
	}

	ctx := types.Context{FormRuleId: "form2"}
	result, err := rules.ProcessForm(ctx, types.KV{})
	require.NoError(t, err)
	assert.Equal(t, types.TextMsg{Text: "second"}, result)
}

func TestRuleset_ProcessForm_EmptyRuleset(t *testing.T) {
	rules := Ruleset{}
	ctx := types.Context{FormRuleId: "form1"}
	result, err := rules.ProcessForm(ctx, types.KV{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRuleset_ProcessForm_LongTermFlag(t *testing.T) {
	rules := Ruleset{
		{
			Id:         "form_lt",
			IsLongTerm: true,
			Title:      "Long Term Form",
			Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
				return types.TextMsg{Text: "long term processed"}
			},
		},
	}

	ctx := types.Context{FormRuleId: "form_lt"}
	result, err := rules.ProcessForm(ctx, types.KV{})
	require.NoError(t, err)
	assert.Equal(t, types.TextMsg{Text: "long term processed"}, result)

	// Verify IsLongTerm is accessible
	assert.True(t, rules[0].IsLongTerm)
}

func TestRuleset_ProcessForm_WithFormFields(t *testing.T) {
	rules := Ruleset{
		{
			Id:    "form_fields",
			Title: "Multi Field Form",
			Field: []types.FormField{
				{Type: types.FormFieldText, Key: "username", Label: "Username"},
				{Type: types.FormFieldPassword, Key: "password", Label: "Password"},
				{Type: types.FormFieldNumber, Key: "age", Label: "Age"},
			},
			Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
				username, _ := values.String("username")
				return types.TextMsg{Text: "User: " + username}
			},
		},
	}

	assert.Len(t, rules[0].Field, 3)
	assert.Equal(t, types.FormFieldText, rules[0].Field[0].Type)
	assert.Equal(t, types.FormFieldPassword, rules[0].Field[1].Type)
	assert.Equal(t, types.FormFieldNumber, rules[0].Field[2].Type)
}
