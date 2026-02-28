package cloudflare

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 2)
}

func TestCommandRules_Defines(t *testing.T) {
	defines := make(map[string]string)
	for _, r := range commandRules {
		defines[r.Define] = r.Help
	}

	assert.Contains(t, defines, "cloudflare setting")
	assert.Contains(t, defines, "cloudflare test")
}

func TestCommandRules_Handlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}
