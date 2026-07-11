package pipeline

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	abilityagent "github.com/flowline-io/flowbot/pkg/ability/agent"
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
	stub := &stubAgentRunner{reply: "agent summary"}
	abilityagent.SetRunner(stub)
	require.NoError(t, abilityagent.Register())
	t.Cleanup(func() {
		abilityagent.SetRunner(nil)
		ability.UnregisterInvoker(hub.CapAgent, ability.OpAgentRun)
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

	res, err := ability.Invoke(context.Background(), hub.CapAgent, ability.OpAgentRun, rendered)
	require.NoError(t, err)

	stepOutput := StepResultFromInvoke(res)
	rc.RecordStepResult("summarize", stepOutput)

	next, err := rc.RenderString(`{{step "summarize" "reply"}}`)
	require.NoError(t, err)
	assert.Equal(t, "agent summary", next)
	assert.Equal(t, "Summarize https://example.com/bookmark titled Example Title", stub.lastPrompt)
}
