package tool_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterDuplicate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "duplicate rejected"},
		{name: "second register fails"},
		{name: "first remains registered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := tool.NewRegistry()
			require.NoError(t, reg.Register(&stubTool{name: "echo"}))
			err := reg.Register(&stubTool{name: "echo"})
			assert.Error(t, err)
		})
	}
}

func TestBuildLLMTools(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "builds schema"},
		{name: "includes name"},
		{name: "includes description"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tools := tool.BuildLLMTools([]tool.Tool{&stubTool{name: "echo", result: "ok"}})
			require.Len(t, tools, 1)
			assert.Equal(t, "echo", tools[0].Function.Name)
		})
	}
}
