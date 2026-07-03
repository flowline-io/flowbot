package tool_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type namedStub struct{ name string }

func (n namedStub) Name() string        { return n.name }
func (n namedStub) Description() string { return "stub tool " + n.name }
func (n namedStub) Parameters() map[string]any {
	return map[string]any{"type": "object", "title": n.name}
}
func (n namedStub) Execute(_ context.Context, _ string, _ map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	_ = n.name
	return msg.ToolResultMessage{}, nil
}

func TestRegistryActiveToolsSorted(t *testing.T) {
	tests := []struct {
		name   string
		active []string
		want   []string
	}{
		{
			name:   "all registered tools sorted",
			active: nil,
			want:   []string{"alpha", "beta", "zebra"},
		},
		{
			name:   "active allowlist sorted",
			active: []string{"zebra", "alpha"},
			want:   []string{"alpha", "zebra"},
		},
		{
			name:   "single active tool",
			active: []string{"beta"},
			want:   []string{"beta"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := tool.NewRegistry()
			for _, name := range []string{"zebra", "alpha", "beta"} {
				require.NoError(t, reg.Register(namedStub{name: name}))
			}
			if tt.active != nil {
				reg.SetActive(tt.active)
			}
			got := reg.ActiveTools()
			names := make([]string, 0, len(got))
			for _, item := range got {
				names = append(names, item.Name())
			}
			assert.Equal(t, tt.want, names)
		})
	}
}
