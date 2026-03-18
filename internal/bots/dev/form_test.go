package dev

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormRules_Count(t *testing.T) {
	assert.Len(t, formRules, 1)
}

func TestFormRules_ID(t *testing.T) {
	assert.Equal(t, devFormID, formRules[0].Id)
	assert.Equal(t, "dev_form", devFormID)
}

func TestFormRules_Title(t *testing.T) {
	assert.NotEmpty(t, formRules[0].Title)
}

func TestFormRules_FieldCount(t *testing.T) {
	assert.Len(t, formRules[0].Field, 8)
}

func TestFormRules_FieldTypes(t *testing.T) {
	expectedTypes := []types.FormFieldType{
		types.FormFieldText,
		types.FormFieldPassword,
		types.FormFieldNumber,
		types.FormFieldRadio,
		types.FormFieldCheckbox,
		types.FormFieldTextarea,
		types.FormFieldSelect,
		types.FormFieldRange,
	}
	for i, f := range formRules[0].Field {
		assert.Equal(t, expectedTypes[i], f.Type, "field %d type mismatch", i)
	}
}

func TestFormRules_Handler(t *testing.T) {
	assert.NotNil(t, formRules[0].Handler)
}

func TestFormRules_Comprehensive(t *testing.T) {
	for _, r := range formRules {
		t.Run(r.Id, func(t *testing.T) {
			assert.NotEmpty(t, r.Id)
			assert.NotEmpty(t, r.Title)
			assert.NotNil(t, r.Handler)
			assert.NotEmpty(t, r.Field)
		})
	}
}

func TestFormRules_AllFieldsHaveRequiredProperties(t *testing.T) {
	for _, r := range formRules {
		for i, f := range r.Field {
			t.Run(r.Id+"_field_"+f.Key, func(t *testing.T) {
				assert.NotEmpty(t, f.Key, "field %d should have key", i)
				assert.NotEmpty(t, f.Type, "field %d should have type", i)
				assert.NotEmpty(t, f.Label, "field %d should have label", i)
			})
		}
	}
}

func TestFormRules_HandlerExecution(t *testing.T) {
	var devFormRule *form.Rule
	for i := range formRules {
		if formRules[i].Id == devFormID {
			devFormRule = &formRules[i]
			break
		}
	}
	require.NotNil(t, devFormRule)
	require.NotNil(t, devFormRule.Handler)

	tests := []struct {
		name         string
		values       types.KV
		wantMsgType  string
		wantContains string
	}{
		{
			name:         "empty values",
			values:       types.KV{},
			wantMsgType:  "TextMsg",
			wantContains: "ok",
		},
		{
			name: "with text value",
			values: types.KV{
				"text": "hello",
			},
			wantMsgType:  "TextMsg",
			wantContains: "ok",
		},
		{
			name: "with multiple values",
			values: types.KV{
				"text":     "hello",
				"password": "secret",
				"number":   42,
			},
			wantMsgType:  "TextMsg",
			wantContains: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.Context{
				Platform: "test",
				Topic:    "test",
				AsUser:   types.Uid("test_user"),
				FormId:   "test_form_id",
			}

			payload := devFormRule.Handler(ctx, tt.values)
			require.NotNil(t, payload)
			assert.Equal(t, tt.wantMsgType, types.TypeOf(payload))

			msg, ok := payload.(types.TextMsg)
			require.True(t, ok)
			assert.Contains(t, msg.Text, tt.wantContains)
			assert.Contains(t, msg.Text, ctx.FormId)
		})
	}
}

func TestFormRuleset_ProcessForm(t *testing.T) {
	rs := form.Ruleset(formRules)
	ctx := types.Context{
		Platform:   "test",
		Topic:      "test",
		AsUser:     types.Uid("test_user"),
		FormRuleId: devFormID,
		FormId:     "test_form_id",
	}

	values := types.KV{
		"text":     "hello",
		"password": "secret",
	}

	payload, err := rs.ProcessForm(ctx, values)
	require.NoError(t, err)
	require.NotNil(t, payload)

	msg, ok := payload.(types.TextMsg)
	require.True(t, ok)
	assert.Contains(t, msg.Text, "ok")
}

func TestFormRuleset_ProcessForm_NotFound(t *testing.T) {
	rs := form.Ruleset(formRules)
	ctx := types.Context{
		Platform:   "test",
		Topic:      "test",
		AsUser:     types.Uid("test_user"),
		FormRuleId: "nonexistent_form",
		FormId:     "test_form_id",
	}

	values := types.KV{}

	payload, err := rs.ProcessForm(ctx, values)
	require.NoError(t, err)
	assert.Nil(t, payload)
}

func TestFormRules_FieldsValidation(t *testing.T) {
	for _, r := range formRules {
		for _, f := range r.Field {
			switch f.Type {
			case types.FormFieldText, types.FormFieldPassword, types.FormFieldTextarea:
				assert.Equal(t, types.FormFieldValueString, f.ValueType)
			case types.FormFieldNumber, types.FormFieldRange:
				assert.Equal(t, types.FormFieldValueInt64, f.ValueType)
			case types.FormFieldRadio, types.FormFieldCheckbox:
				if f.Type == types.FormFieldRadio {
					assert.Equal(t, types.FormFieldValueBool, f.ValueType)
				}
			case types.FormFieldSelect:
				assert.Equal(t, types.FormFieldValueFloat64, f.ValueType)
			}
		}
	}
}
