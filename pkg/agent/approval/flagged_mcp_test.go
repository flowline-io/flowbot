package approval_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateFlaggedMCP(t *testing.T) {
	t.Parallel()
	permission.SetMCPToolMeta("mcp_ha_get_state", true, false)
	t.Cleanup(func() { permission.ClearMCPToolMeta("mcp_ha_get_state") })
	assert.True(t, approval.IsReadonlyTool("mcp_ha_get_state"))
	got := approval.EvaluateFlagged(permission.Request{Tool: "mcp_ha_get_state"})
	assert.False(t, got.Flagged)

	permission.SetMCPToolMeta("mcp_ha_turn_off", false, true)
	t.Cleanup(func() { permission.ClearMCPToolMeta("mcp_ha_turn_off") })
	got = approval.EvaluateFlagged(permission.Request{Tool: "mcp_ha_turn_off"})
	assert.True(t, got.Flagged)
	assert.Equal(t, "mcp destructive tool", got.Reason)

	permission.SetMCPToolMeta("mcp_ha_set_state", false, false)
	t.Cleanup(func() { permission.ClearMCPToolMeta("mcp_ha_set_state") })
	got = approval.EvaluateFlagged(permission.Request{Tool: "mcp_ha_set_state"})
	assert.True(t, got.Flagged)
	assert.Equal(t, "mcp write tool", got.Reason)
}
