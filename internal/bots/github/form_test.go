package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormRules_Empty(t *testing.T) {
	assert.Empty(t, formRules)
}
