package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolRules_Count(t *testing.T) {
	assert.Len(t, toolRules, 2)
}

func TestToolRules_NotNil(t *testing.T) {
	for i, r := range toolRules {
		assert.NotNil(t, r, "tool rule %d should not be nil", i)
	}
}
