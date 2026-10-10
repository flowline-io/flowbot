package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
)

func TestMCPPermissionKeyAndDefaults(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "mcp.ha", permission.PermissionKeyForTool("mcp_ha_light.turn_on"))
	permission.SetMCPToolMeta("mcp_ha_status", true, false)
	t.Cleanup(func() { permission.ClearMCPToolMeta("mcp_ha_status") })
	eval := permission.NewEvaluator(permission.DefaultConfig())
	got := eval.Evaluate(permission.Request{Tool: "mcp_ha_status"}, nil)
	assert.Equal(t, permission.ActionAllow, got.Action)
	assert.Equal(t, "mcp.ha", got.PermissionKey)

	permission.SetMCPToolMeta("mcp_ha_turn_on", false, true)
	t.Cleanup(func() { permission.ClearMCPToolMeta("mcp_ha_turn_on") })
	got = eval.Evaluate(permission.Request{Tool: "mcp_ha_turn_on"}, nil)
	assert.Equal(t, permission.ActionAsk, got.Action)
}
