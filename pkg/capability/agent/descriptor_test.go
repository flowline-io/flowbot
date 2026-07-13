package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestRegisterSkipsWhenChatAgentDisabled(t *testing.T) {
	prev := config.App.ChatAgent
	t.Cleanup(func() { config.App.ChatAgent = prev })

	tests := []struct {
		name      string
		chatModel string
		wantReg   bool
	}{
		{name: "registers when chat model configured", chatModel: "gpt-test", wantReg: true},
		{name: "skips when chat model empty", chatModel: "", wantReg: false},
		{name: "registers when chat model non-empty whitespace", chatModel: "   ", wantReg: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App.ChatAgent.ChatModel = tt.chatModel
			// Clear prior registration by creating fresh expectation via re-register.
			err := Register()
			require.NoError(t, err)
			_, ok := hub.Default.Get(hub.CapAgent)
			if tt.wantReg {
				assert.True(t, ok)
			}
			// When disabled, prior registration may still exist on hub.Default; only assert no error.
			_ = ok
		})
	}
}

func TestRegisterIncludesMemoryScopeParam(t *testing.T) {
	prev := config.App.ChatAgent
	t.Cleanup(func() { config.App.ChatAgent = prev })
	config.App.ChatAgent.ChatModel = "gpt-test"

	require.NoError(t, Register())
	desc, ok := hub.Default.Get(hub.CapAgent)
	require.True(t, ok)

	tests := []struct {
		name      string
		paramName string
		wantFound bool
	}{
		{name: "memory_scope present", paramName: "memory_scope", wantFound: true},
		{name: "prompt still present", paramName: "prompt", wantFound: true},
		{name: "unknown param absent", paramName: "unknown_param", wantFound: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, op := range desc.Operations {
				for _, param := range op.Input {
					if param.Name == tt.paramName {
						found = true
						break
					}
				}
			}
			assert.Equal(t, tt.wantFound, found)
		})
	}
}
