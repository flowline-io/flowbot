package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/pkg/ability"
	abilityagent "github.com/flowline-io/flowbot/pkg/ability/agent"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type webStubAgentRunner struct {
	replies []string
	prompts []string
	calls   int
}

type pipelineTestStepResult struct {
	Name   string         `json:"name"`
	Status string         `json:"status"`
	Output map[string]any `json:"output"`
}

type pipelineTestStepResponse struct {
	Success bool                     `json:"success"`
	Steps   []pipelineTestStepResult `json:"steps"`
}

func (s *webStubAgentRunner) Run(_ context.Context, params abilityagent.RunParams) (*abilityagent.RunResult, error) {
	s.prompts = append(s.prompts, params.Prompt)
	reply := "agent-reply"
	if s.calls < len(s.replies) {
		reply = s.replies[s.calls]
	}
	s.calls++
	return &abilityagent.RunResult{Reply: reply, SessionID: "sess-web-mock"}, nil
}

func registerWebAgentRunner(t *testing.T, runner abilityagent.Runner) {
	t.Helper()
	abilityagent.SetRunner(runner)
	require.NoError(t, abilityagent.Register())
	t.Cleanup(func() {
		abilityagent.SetRunner(nil)
		ability.UnregisterInvoker(hub.CapAgent, ability.OpAgentRun)
		hub.Default.Unregister(hub.CapAgent)
	})
}

func TestTestPipelineStepAgentMultiStepOutput(t *testing.T) {
	stub := &webStubAgentRunner{replies: []string{"first-agent-reply", "second-agent-reply"}}
	registerWebAgentRunner(t, stub)

	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ps := store.NewPipelineStore(client)
	require.NoError(t, ps.CreateDefinition(ctx, "agent-chain-test", ""))
	yamlDraft := `name: agent-chain-test
steps:
  - name: summarize
    capability: agent
    operation: run
    params:
      prompt: "summarize {{.Event.url}}"
  - name: follow-up
    capability: agent
    operation: run
    params:
      prompt: "Previous reply: {{step \"summarize\" \"reply\"}}"`
	require.NoError(t, client.PipelineDefinition.Update().
		SetYamlDraft(yamlDraft).
		Where(pipelinedefinition.Name("agent-chain-test")).
		Exec(ctx))

	body, err := sonic.MarshalString(map[string]any{
		"trigger_source":   "event",
		"mock_payload":     map[string]any{"url": "https://example.com/item"},
		"up_to_step_index": 1,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/service/web/pipelines/agent-chain-test/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var payload pipelineTestStepResponse
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.True(t, payload.Success)
	require.Len(t, payload.Steps, 2)
	assert.Equal(t, "ok", payload.Steps[0].Status)
	assert.Equal(t, "first-agent-reply", payload.Steps[0].Output["reply"])
	assert.Equal(t, "ok", payload.Steps[1].Status)
	assert.Equal(t, "second-agent-reply", payload.Steps[1].Output["reply"])
	require.Len(t, stub.prompts, 2)
	assert.Equal(t, "summarize https://example.com/item", stub.prompts[0])
	assert.Equal(t, "Previous reply: first-agent-reply", stub.prompts[1])
}

func TestTestPipelineStepRejectsWhitespaceAgentPrompt(t *testing.T) {
	registerWebAgentRunner(t, &webStubAgentRunner{})

	app, _, client := setupTestAppWithDB(t)
	t.Cleanup(func() { store.Database = nil; handler = moduleHandler{}; config = configType{} })

	ctx := context.Background()
	ps := store.NewPipelineStore(client)
	require.NoError(t, ps.CreateDefinition(ctx, "agent-empty-prompt", ""))
	require.NoError(t, client.PipelineDefinition.Update().
		SetYamlDraft(`name: agent-empty-prompt
steps:
  - name: bad
    capability: agent
    operation: run
    params:
      prompt: "   "`).
		Where(pipelinedefinition.Name("agent-empty-prompt")).
		Exec(ctx))

	body := bytes.NewBufferString(`{"trigger_source":"event","mock_payload":{},"up_to_step_index":0}`)
	req := httptest.NewRequest(http.MethodPost, "/service/web/pipelines/agent-empty-prompt/test", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"success":false`)
}
