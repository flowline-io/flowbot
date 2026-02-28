package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 9)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "workflow list")
	assert.Contains(t, defines, "workflow get [id]")
	assert.Contains(t, defines, "workflow create [name]")
	assert.Contains(t, defines, "workflow update [id] [name]")
	assert.Contains(t, defines, "workflow delete [id]")
	assert.Contains(t, defines, "workflow activate [id]")
	assert.Contains(t, defines, "workflow deactivate [id]")
	assert.Contains(t, defines, "workflow execute [id]")
	assert.Contains(t, defines, "workflow stat")
}

func TestCommandRules_Handlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}
