package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectRules_Count(t *testing.T) {
	assert.Len(t, collectRules, 1)
}

func TestCollectRules_ImportCollectID(t *testing.T) {
	assert.Equal(t, "import_collect", ImportCollectID)
	assert.Equal(t, ImportCollectID, collectRules[0].Id)
}

func TestCollectRules_Handler(t *testing.T) {
	assert.NotNil(t, collectRules[0].Handler)
}
