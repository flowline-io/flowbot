package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDenied(t *testing.T) {
	t.Parallel()
	assert.True(t, Denied("core", "run_terminal"))
	assert.True(t, Denied("core", "agent_run"))
	assert.True(t, Denied("gateway", "run"))
	assert.False(t, Denied("core", "notify_send"))
	assert.False(t, Denied("karakeep", "create"))
}
