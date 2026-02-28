package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 8)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "github setting")
	assert.Contains(t, defines, "github oauth")
	assert.Contains(t, defines, "github user")
	assert.Contains(t, defines, "github issue [string]")
	assert.Contains(t, defines, "github card [string]")
	assert.Contains(t, defines, "github repo [string]")
	assert.Contains(t, defines, "github user [string]")
	assert.Contains(t, defines, "deploy")
}

func TestCommandRules_Handlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}
