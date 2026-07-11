package agent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestDescriptorHealthyReflectsChatAgentEnabled(t *testing.T) {
	prev := config.App.ChatAgent
	t.Cleanup(func() { config.App.ChatAgent = prev })

	tests := []struct {
		name        string
		chatModel   string
		wantHealthy bool
	}{
		{name: "enabled when chat model configured", chatModel: "gpt-test", wantHealthy: true},
		{name: "disabled when chat model empty", chatModel: "", wantHealthy: false},
		{name: "enabled when chat model non-empty whitespace", chatModel: "   ", wantHealthy: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.App.ChatAgent.ChatModel = tt.chatModel
			assert.Equal(t, tt.wantHealthy, Descriptor().Healthy)
		})
	}
}

func TestDescriptorIncludesMemoryScopeParam(t *testing.T) {
	prev := config.App.ChatAgent
	t.Cleanup(func() { config.App.ChatAgent = prev })
	config.App.ChatAgent.ChatModel = "gpt-test"

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
			desc := Descriptor()
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
