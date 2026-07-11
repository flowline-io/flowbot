package server

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAgentAbilityRegistersInvoker(t *testing.T) {
	ability.UnregisterInvoker(hub.CapAgent, ability.OpAgentRun)
	hub.Default.Unregister(hub.CapAgent)

	require.NoError(t, initAgentAbility())
	t.Cleanup(func() {
		ability.UnregisterInvoker(hub.CapAgent, ability.OpAgentRun)
		hub.Default.Unregister(hub.CapAgent)
	})

	_, err := ability.Invoke(t.Context(), hub.CapAgent, ability.OpAgentRun, map[string]any{
		"prompt": "",
	})
	assert.Error(t, err)
}
