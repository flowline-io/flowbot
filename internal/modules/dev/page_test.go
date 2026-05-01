package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageRules_Count(t *testing.T) {
	assert.Len(t, pageRules, 1)
}

func TestPageRules_ID(t *testing.T) {
	assert.Equal(t, "dev", pageRules[0].Id)
}

func TestPageRules_UI(t *testing.T) {
	assert.NotNil(t, pageRules[0].UI)
}
