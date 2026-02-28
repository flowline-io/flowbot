package finance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCronRules_Count(t *testing.T) {
	assert.Len(t, cronRules, 1)
}

func TestCronRules_Name(t *testing.T) {
	assert.Equal(t, "finance_example", cronRules[0].Name)
}

func TestCronRules_When(t *testing.T) {
	assert.Equal(t, "* * * * *", cronRules[0].When)
}

func TestCronRules_Action(t *testing.T) {
	assert.NotNil(t, cronRules[0].Action)
}
