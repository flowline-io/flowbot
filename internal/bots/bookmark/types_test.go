package bookmark

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagPrompt_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, tagPrompt)
}

func TestTagPrompt_ContainsRequiredSections(t *testing.T) {
	assert.Contains(t, tagPrompt, "tags")
	assert.Contains(t, tagPrompt, "JSON")
	assert.Contains(t, tagPrompt, "{{.language}}")
}
