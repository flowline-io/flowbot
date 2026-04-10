package finance

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestWallosWebhookData_EmptyFields(t *testing.T) {
	d := wallosWebhookData{}
	assert.Empty(t, d.Event)
	assert.Empty(t, d.Amount)
	assert.Empty(t, d.Currency)
	assert.Empty(t, d.Category)
	assert.Empty(t, d.Payee)
	assert.Empty(t, d.Notes)
	assert.Empty(t, d.Date)
	assert.Empty(t, d.Account)
	assert.False(t, d.Recurring)
}

func TestWallosWebhookData_WithCurrency(t *testing.T) {
	d := wallosWebhookData{
		Event:    "subscription_created",
		Amount:   "9.99",
		Currency: "EUR",
	}
	assert.Equal(t, "EUR", d.Currency)
}

func TestImportBillFormID(t *testing.T) {
	assert.Equal(t, "import_bill", importBillFormID)
}

func TestFormRulesFieldType(t *testing.T) {
	f := formRules[0].Field[0]
	assert.Equal(t, "bill", f.Key)
	assert.Equal(t, types.FormFieldTextarea, f.Type)
	assert.Equal(t, types.FormFieldValueString, f.ValueType)
	assert.Equal(t, "required", f.Rule)
}

func TestCronRulesDetails(t *testing.T) {
	assert.Len(t, cronRules, 1)
	assert.Equal(t, "finance_example", cronRules[0].Name)
	assert.Equal(t, "* * * * *", cronRules[0].When)
}

func TestCommandRulesDetails(t *testing.T) {
	assert.Len(t, commandRules, 1)
	assert.Equal(t, "bill", commandRules[0].Define)
	assert.Equal(t, "Import bill", commandRules[0].Help)
}
