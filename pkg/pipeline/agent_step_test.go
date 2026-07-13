package pipeline

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/capability"
	abilityagent "github.com/flowline-io/flowbot/pkg/capability/agent"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAgentRunner struct {
	lastPrompt string
	reply      string
}

func (s *stubAgentRunner) Run(_ context.Context, params abilityagent.RunParams) (*abilityagent.RunResult, error) {
	s.lastPrompt = params.Prompt
	return &abilityagent.RunResult{Reply: s.reply, SessionID: "sess-mock"}, nil
}

func TestAgentStepTemplateAndInvoke(t *testing.T) {
	prevModel := config.App.ChatAgent.ChatModel
	config.App.ChatAgent.ChatModel = "gpt-test"
	t.Cleanup(func() { config.App.ChatAgent.ChatModel = prevModel })

	stub := &stubAgentRunner{reply: "agent summary"}
	abilityagent.SetRunner(stub)
	require.NoError(t, abilityagent.Register())
	t.Cleanup(func() {
		abilityagent.SetRunner(nil)
		capability.UnregisterInvoker(hub.CapAgent, capability.OpAgentRun)
		hub.Default.Unregister(hub.CapAgent)
	})

	event := types.DataEvent{
		EventID:   "evt-agent",
		EventType: types.EventBookmarkCreated,
		UID:       "user-7",
		Data:      types.KV{"url": "https://example.com/bookmark"},
	}
	rc := NewRenderContext(event)
	rc.RecordStepResult("fetch-meta", map[string]any{"title": "Example Title"})

	rendered, err := rc.RenderParams(map[string]any{
		"prompt": "Summarize {{.Event.url}} titled {{step \"fetch-meta\" \"title\"}}",
		"uid":    "{{.Event.uid}}",
	})
	require.NoError(t, err)

	res, err := capability.Invoke(context.Background(), hub.CapAgent, capability.OpAgentRun, rendered)
	require.NoError(t, err)

	stepOutput := StepResultFromInvoke(res)
	rc.RecordStepResult("summarize", stepOutput)

	next, err := rc.RenderString(`{{step "summarize" "reply"}}`)
	require.NoError(t, err)
	assert.Equal(t, "agent summary", next)
	assert.Equal(t, "Summarize https://example.com/bookmark titled Example Title", stub.lastPrompt)
}
