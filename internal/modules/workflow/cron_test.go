package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCronRules_Empty(t *testing.T) {
	assert.Empty(t, cronRules)
}
