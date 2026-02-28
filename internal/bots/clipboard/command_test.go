package clipboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRules_Count(t *testing.T) {
	assert.Len(t, commandRules, 1)
}

func TestCommandRules_Defines(t *testing.T) {
	assert.Equal(t, "share [string]", commandRules[0].Define)
	assert.Equal(t, "share clipboard to agent", commandRules[0].Help)
}

func TestCommandRules_Handlers(t *testing.T) {
	for _, r := range commandRules {
		assert.NotNil(t, r.Handler, "handler for %q should not be nil", r.Define)
	}
}
