package finance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBillPrompt_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, billPrompt)
}

func TestBillPrompt_ContainsRequiredContent(t *testing.T) {
	assert.Contains(t, billPrompt, "JSON")
	assert.Contains(t, billPrompt, "date")
	assert.Contains(t, billPrompt, "amount")
	assert.Contains(t, billPrompt, "merchant")
}
