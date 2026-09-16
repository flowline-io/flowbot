package browser_test

import (
	"context"
	"testing"

	agentbrowser "github.com/flowline-io/flowbot/pkg/agent/browser"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	toolbrowser "github.com/flowline-io/flowbot/pkg/agent/tools/browser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAndActiveNames(t *testing.T) {
	t.Parallel()
	reg := tool.NewRegistry()
	require.NoError(t, toolbrowser.Register(reg))
	for _, name := range toolbrowser.ActiveToolNames() {
		_, ok := reg.Get(name)
		assert.True(t, ok, name)
	}
}

func TestNavigateWithoutSession(t *testing.T) {
	t.Parallel()
	res, err := toolbrowser.NavigateTool{}.Execute(context.Background(), "1", map[string]any{"url": "https://example.com"}, nil)
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestNavigateWithSessionURLGate(t *testing.T) {
	t.Parallel()
	sess, err := agentbrowser.NewSession(agentbrowser.Config{})
	require.NoError(t, err)
	ctx := agentbrowser.WithSession(context.Background(), sess)
	res, err := toolbrowser.NavigateTool{}.Execute(ctx, "1", map[string]any{"url": "http://127.0.0.1/"}, nil)
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, textOf(res), "not allowed")
}

func textOf(res msg.ToolResultMessage) string {
	for _, p := range res.Parts {
		if tp, ok := p.(msg.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}
