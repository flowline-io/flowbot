package finance

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestFormConstants(t *testing.T) {
	assert.Equal(t, "import_bill", importBillFormID)
}

func TestFormRules_Count(t *testing.T) {
	assert.Len(t, formRules, 1)
}

func TestFormRules_ID(t *testing.T) {
	assert.Equal(t, importBillFormID, formRules[0].Id)
}

func TestFormRules_Title(t *testing.T) {
	assert.Equal(t, "Import Bill", formRules[0].Title)
}

func TestFormRules_Fields(t *testing.T) {
	assert.Len(t, formRules[0].Field, 1)
	f := formRules[0].Field[0]
	assert.Equal(t, "bill", f.Key)
	assert.Equal(t, types.FormFieldTextarea, f.Type)
	assert.Equal(t, types.FormFieldValueString, f.ValueType)
	assert.Equal(t, "required", f.Rule)
}

func TestFormRules_Handler(t *testing.T) {
	assert.NotNil(t, formRules[0].Handler)
}
