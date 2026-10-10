package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerNames(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidServerName("ha"))
	assert.True(t, ValidServerName("home-assistant"))
	assert.False(t, ValidServerName("Home"))
	assert.False(t, ValidServerName("ha_mcp"))
	assert.False(t, ValidServerName("1ha"))
	require.NoError(t, ValidateServerName("ha"))
	require.Error(t, ValidateServerName("ha_mcp"))
}

func TestAgentToolName(t *testing.T) {
	t.Parallel()
	name := AgentToolName("ha", "light.turn_on")
	assert.Equal(t, "mcp_ha_light.turn_on", name)
	server, ok := ServerFromAgentTool(name)
	assert.True(t, ok)
	assert.Equal(t, "ha", server)
	assert.Equal(t, "mcp.ha", PermissionKey("ha"))
	_, ok = ServerFromAgentTool("send_notification")
	assert.False(t, ok)
	assert.True(t, IsAgentTool(name))
	assert.False(t, IsAgentTool("send_notification"))
}
