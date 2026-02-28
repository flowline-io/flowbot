package dev

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
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
