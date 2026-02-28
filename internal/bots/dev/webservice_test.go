package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebserviceRules_Count(t *testing.T) {
	assert.Len(t, webserviceRules, 1)
}

func TestWebserviceRules_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, webserviceRules)
}
