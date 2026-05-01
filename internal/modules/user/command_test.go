package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRules_Empty(t *testing.T) {
	assert.Empty(t, commandRules)
}
