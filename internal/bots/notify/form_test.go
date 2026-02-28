package notify

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestFormConstants(t *testing.T) {
	assert.Equal(t, "create_notify", createNotifyFormID)
}

func TestFormRules_Count(t *testing.T) {
	assert.Len(t, formRules, 1)
}

func TestFormRules_ID(t *testing.T) {
	assert.Equal(t, createNotifyFormID, formRules[0].Id)
}

func TestFormRules_Title(t *testing.T) {
	assert.Equal(t, "Create one task", formRules[0].Title)
}

func TestFormRules_IsLongTerm(t *testing.T) {
	assert.True(t, formRules[0].IsLongTerm)
}

func TestFormRules_Fields(t *testing.T) {
	assert.Len(t, formRules[0].Field, 2)

	nameField := formRules[0].Field[0]
	assert.Equal(t, "name", nameField.Key)
	assert.Equal(t, types.FormFieldText, nameField.Type)
	assert.Equal(t, "required", nameField.Rule)

	templateField := formRules[0].Field[1]
	assert.Equal(t, "template", templateField.Key)
	assert.Equal(t, types.FormFieldTextarea, templateField.Type)
	assert.Equal(t, "required", templateField.Rule)
}

func TestFormRules_Handler(t *testing.T) {
	assert.NotNil(t, formRules[0].Handler)
}
