package server

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/core"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAgentAbilityRegistersInvoker(t *testing.T) {
	capability.UnregisterInvoker(hub.CapCore, capability.OpAgentRun)
	hub.Default.Unregister(hub.CapCore)

	require.NoError(t, initAgentAbility())
	require.NoError(t, core.Register())
	t.Cleanup(func() {
		capability.UnregisterInvoker(hub.CapCore, capability.OpAgentRun)
		hub.Default.Unregister(hub.CapCore)
	})

	_, err := capability.Invoke(t.Context(), hub.CapCore, capability.OpAgentRun, map[string]any{
		"prompt": "",
	})
	assert.Error(t, err)
}
