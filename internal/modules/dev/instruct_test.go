package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstructRules_Count(t *testing.T) {
	assert.Len(t, instructRules, 1)
}

func TestInstructRules_ID(t *testing.T) {
	assert.Equal(t, ExampleInstructID, instructRules[0].Id)
	assert.Equal(t, "dev_example", ExampleInstructID)
}

func TestInstructRules_Args(t *testing.T) {
	assert.Equal(t, []string{"txt"}, instructRules[0].Args)
}
