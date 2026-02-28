package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectConstants(t *testing.T) {
	assert.Equal(t, "search_example_collect", ExampleCollectID)
}

func TestCollectRules_Count(t *testing.T) {
	assert.Len(t, collectRules, 1)
}

func TestCollectRules_ID(t *testing.T) {
	assert.Equal(t, ExampleCollectID, collectRules[0].Id)
}

func TestCollectRules_Handler(t *testing.T) {
	assert.NotNil(t, collectRules[0].Handler)
}
