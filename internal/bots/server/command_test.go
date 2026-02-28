package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 10)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "version")
	assert.Contains(t, defines, "mem stats")
	assert.Contains(t, defines, "golang stats")
	assert.Contains(t, defines, "server stats")
	assert.Contains(t, defines, "online stats")
	assert.Contains(t, defines, "instruct list")
	assert.Contains(t, defines, "adguard status")
	assert.Contains(t, defines, "adguard stats")
	assert.Contains(t, defines, "queue stats")
	assert.Contains(t, defines, "check")
}

func TestCommandRules_Handlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}
