package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 1)
}

func TestCommandRules_Defines(t *testing.T) {
	assert.Equal(t, "search", commandRules[0].Define)
	assert.Equal(t, "Search page", commandRules[0].Help)
}

func TestCommandRules_Handler(t *testing.T) {
	assert.NotNil(t, commandRules[0].Handler)
}
