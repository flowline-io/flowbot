package server

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAgentAbilityRegistersInvoker(t *testing.T) {
	capability.UnregisterInvoker(hub.CapAgent, capability.OpAgentRun)
	hub.Default.Unregister(hub.CapAgent)

	require.NoError(t, initAgentAbility())
	t.Cleanup(func() {
		capability.UnregisterInvoker(hub.CapAgent, capability.OpAgentRun)
		hub.Default.Unregister(hub.CapAgent)
	})

	_, err := capability.Invoke(t.Context(), hub.CapAgent, capability.OpAgentRun, map[string]any{
		"prompt": "",
	})
	assert.Error(t, err)
}
