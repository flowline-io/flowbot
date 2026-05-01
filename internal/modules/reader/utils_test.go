package reader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAIResult_FunctionExists(t *testing.T) {
	// getAIResult depends on external LLM services, so we only verify the function exists
	assert.NotNil(t, getAIResult)
}
