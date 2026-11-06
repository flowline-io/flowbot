package core_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/core"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

type stubAgentRunner struct {
	called bool
}

func (s *stubAgentRunner) Run(_ context.Context, _ core.RunParams) (*core.RunResult, error) {
	s.called = true
	return &core.RunResult{Reply: "ok", SessionID: "sess"}, nil
}

func TestAgentRunUnavailableWhenChatAgentDisabled(t *testing.T) {
	prev := config.App.ChatAgent
	enabledFalse := false
	config.App.ChatAgent = config.ChatAgentConfig{
		Enabled:   &enabledFalse,
		ChatModel: "gpt-test",
	}
	t.Cleanup(func() { config.App.ChatAgent = prev })

	stub := &stubAgentRunner{}
	core.SetRunner(stub)
	resetCore(t)
	t.Cleanup(func() {
		core.SetRunner(nil)
		capability.UnregisterInvoker(hub.CapCore, capability.OpAgentRun)
	})

	_, err := capability.Invoke(context.Background(), hub.CapCore, core.OpAgentRun, map[string]any{
		"prompt": "hello",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrUnavailable)
	assert.Contains(t, err.Error(), "chat agent is disabled")
	assert.False(t, stub.called)

	_, err = capability.Invoke(context.Background(), hub.CapCore, core.OpAgentHealth, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrUnavailable)
	assert.Contains(t, err.Error(), "chat agent is disabled")
}

func TestAgentRunInvokesWhenEnabled(t *testing.T) {
	prev := config.App.ChatAgent
	enabledTrue := true
	config.App.ChatAgent = config.ChatAgentConfig{
		Enabled:   &enabledTrue,
		ChatModel: "gpt-test",
	}
	t.Cleanup(func() { config.App.ChatAgent = prev })

	stub := &stubAgentRunner{}
	core.SetRunner(stub)
	resetCore(t)
	t.Cleanup(func() {
		core.SetRunner(nil)
		capability.UnregisterInvoker(hub.CapCore, capability.OpAgentRun)
	})

	res, err := capability.Invoke(context.Background(), hub.CapCore, core.OpAgentRun, map[string]any{
		"prompt": "hello",
	})
	require.NoError(t, err)
	assert.True(t, stub.called)
	data, ok := res.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", data["reply"])
}
