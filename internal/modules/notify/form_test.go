package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestFormConstants(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "create_notify form id constant",
			fn: func(t *testing.T) {
				assert.Equal(t, "create_notify", createNotifyFormID)
			},
		},
		{
			name: "one form rule",
			fn: func(t *testing.T) {
				assert.Len(t, formRules, 1)
			},
		},
		{
			name: "form rule id matches constant",
			fn: func(t *testing.T) {
				assert.Equal(t, createNotifyFormID, formRules[0].Id)
			},
		},
		{
			name: "form rule has title",
			fn: func(t *testing.T) {
				assert.Equal(t, "Create one task", formRules[0].Title)
			},
		},
		{
			name: "form rule is long term",
			fn: func(t *testing.T) {
				assert.True(t, formRules[0].IsLongTerm)
			},
		},
		{
			name: "form rule has two fields",
			fn: func(t *testing.T) {
				assert.Len(t, formRules[0].Field, 2)

				nameField := formRules[0].Field[0]
				assert.Equal(t, "name", nameField.Key)
				assert.Equal(t, types.FormFieldText, nameField.Type)
				assert.Equal(t, "required", nameField.Rule)

				templateField := formRules[0].Field[1]
				assert.Equal(t, "template", templateField.Key)
				assert.Equal(t, types.FormFieldTextarea, templateField.Type)
				assert.Equal(t, "required", templateField.Rule)
			},
		},
		{
			name: "form rule handler not nil",
			fn: func(t *testing.T) {
				assert.NotNil(t, formRules[0].Handler)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}
