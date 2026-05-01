package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowName(t *testing.T) {
	assert.Equal(t, "workflow", Name)
}

func TestCronRulesEmpty(t *testing.T) {
	assert.Empty(t, cronRules)
}

func TestCommandRulesAllHaveHelp(t *testing.T) {
	for _, r := range commandRules {
		assert.NotEmpty(t, r.Help, "command %q should have Help text", r.Define)
	}
}

func TestConfigType(t *testing.T) {
	cfg := configType{Enabled: true}
	assert.True(t, cfg.Enabled)

	cfg = configType{Enabled: false}
	assert.False(t, cfg.Enabled)
}
